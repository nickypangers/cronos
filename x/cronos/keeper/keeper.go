package keeper

import (
	"fmt"

	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	ibctransfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"

	evmTypes "github.com/tharsis/ethermint/x/evm/types"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/crypto-org-chain/cronos/x/cronos/types"
	// this line is used by starport scaffolding # ibc/keeper/import
)

type (
	Keeper struct {
		cdc      codec.Codec
		storeKey sdk.StoreKey
		memKey   sdk.StoreKey

		// module specific parameter space that can be configured through governance
		paramSpace paramtypes.Subspace
		// evm parameter space
		evmParamSpace paramtypes.Subspace
		// update balance and accounting operations with coins
		bankKeeper types.BankKeeper
		// ibc transfer operations
		transferKeeper types.TransferKeeper

		// this line is used by starport scaffolding # ibc/keeper/attribute
	}
)

func NewKeeper(
	cdc codec.Codec,
	storeKey,
	memKey sdk.StoreKey,
	paramSpace paramtypes.Subspace,
	evmSpace paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	transferKeeper types.TransferKeeper,
	// this line is used by starport scaffolding # ibc/keeper/parameter
) *Keeper {

	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	if !evmSpace.HasKeyTable() {
		evmSpace = evmSpace.WithKeyTable(evmTypes.ParamKeyTable())
	}

	return &Keeper{
		cdc:            cdc,
		storeKey:       storeKey,
		memKey:         memKey,
		paramSpace:     paramSpace,
		evmParamSpace:  evmSpace,
		bankKeeper:     bankKeeper,
		transferKeeper: transferKeeper,
		// this line is used by starport scaffolding # ibc/keeper/return
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) ConvertVouchersToEvmCoins(ctx sdk.Context, from string, coins sdk.Coins) error {
	acc, err := sdk.AccAddressFromBech32(from)
	if err != nil {
		return err
	}

	params := k.GetParams(ctx)
	evmParams := k.GetEvmParams(ctx)
	for _, c := range coins {
		switch c.Denom {
		case params.IbcCroDenom:
			if params.IbcCroDenom == "" {
				return sdkerrors.Wrap(types.ErrIbcCroDenomEmpty, "ibc is disabled")
			}

			// Send ibc tokens to escrow address
			err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, sdk.NewCoins(c))
			if err != nil {
				return err
			}
			// Compute new amount, because basecro is a 8 decimals token, we need to multiply by 10^10 to make it
			// a 18 decimals token
			amount18dec := sdk.NewCoin(evmParams.EvmDenom, c.Amount.Mul(sdk.NewIntFromBigInt(types.TenPowTen)))

			// Mint new evm tokens
			if err := k.bankKeeper.MintCoins(
				ctx, types.ModuleName, sdk.NewCoins(amount18dec),
			); err != nil {
				return err
			}

			// Send evm tokens to receiver
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(
				ctx, types.ModuleName, acc, sdk.NewCoins(amount18dec),
			); err != nil {
				return err
			}

		default:
			// TODO handle ERC20 tokens
		}
	}
	defer func() {
		for _, a := range coins {
			if a.Amount.IsInt64() {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "ConvertVouchersToEvmCoins"},
					float32(a.Amount.Int64()),
					[]metrics.Label{telemetry.NewLabel("denom", a.Denom)},
				)
			}
		}
	}()
	return nil
}

func (k Keeper) IbcTransferCoins(ctx sdk.Context, from, destination string, coins sdk.Coins) error {
	acc, err := sdk.AccAddressFromBech32(from)
	if err != nil {
		return err
	}

	params := k.GetParams(ctx)
	evmParams := k.GetEvmParams(ctx)

	for _, c := range coins {
		switch c.Denom {
		case evmParams.EvmDenom:
			// Compute the remainder, we won't transfer anything lower than 10^10
			amount8decRem := c.Amount.Mod(sdk.NewIntFromBigInt(types.TenPowTen))
			amountToBurn := c.Amount.Sub(amount8decRem)
			if amountToBurn.IsZero() {
				// Amount too small
				break
			}
			coins := sdk.NewCoins(sdk.NewCoin(evmParams.EvmDenom, amountToBurn))

			// Send evm tokens to escrow address
			err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, acc, types.ModuleName, coins)
			if err != nil {
				return err
			}
			// Burns the evm tokens
			if err := k.bankKeeper.BurnCoins(
				ctx, types.ModuleName, coins); err != nil {
				return err
			}

			// Transfer ibc tokens back to the user
			// We divide by 10^10 to come back to an 8decimals token
			amount8dec := c.Amount.Quo(sdk.NewIntFromBigInt(types.TenPowTen))
			ibcCoin := sdk.NewCoin(params.IbcCroDenom, amount8dec)
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(
				ctx, types.ModuleName, acc, sdk.NewCoins(ibcCoin),
			); err != nil {
				return err
			}

			channelID, err := k.GetSourceChannelID(ctx, params.IbcCroDenom)
			if err != nil {
				return err
			}
			// Transfer coins to receiver through IBC
			// We use current time for timeout timestamp and zero height for timeoutHeight
			// it means it can never fail by timeout
			// TODO Might need to consider add timeout option in configuration.
			timeoutTimestamp := ctx.BlockTime().UnixNano()
			timeoutHeight := ibcclienttypes.ZeroHeight()
			err = k.transferKeeper.SendTransfer(
				ctx,
				ibctransfertypes.PortID,
				channelID,
				ibcCoin,
				acc,
				destination,
				timeoutHeight,
				uint64(timeoutTimestamp))
			if err != nil {
				return err
			}

		default:
			// TODO handle erc20 tokens
		}
	}

	defer func() {
		for _, a := range coins {
			if a.Amount.IsInt64() {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "IbcTransferCoins"},
					float32(a.Amount.Int64()),
					[]metrics.Label{telemetry.NewLabel("denom", a.Denom)},
				)
			}
		}
	}()
	return nil
}
