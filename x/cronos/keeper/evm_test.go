package keeper_test

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tharsis/ethermint/crypto/ethsecp256k1"
)

func (suite *KeeperTestSuite) TestDeployContract() {
	suite.SetupTest()
	keeper := suite.app.CronosKeeper

	_, err := keeper.DeployCronosERC20(suite.ctx, "test")
	suite.Require().NoError(err)
}

func (suite *KeeperTestSuite) TestTokenConversion() {
	suite.SetupTest()
	keeper := suite.app.CronosKeeper

	// generate test address
	priv, err := ethsecp256k1.GenerateKey()
	suite.Require().NoError(err)
	address := common.BytesToAddress(priv.PubKey().Address().Bytes())

	denom := "testdenom"
	amount := big.NewInt(100)
	coins := sdk.NewCoins(sdk.NewCoin(denom, sdk.NewIntFromBigInt(amount)))

	// mint native tokens
	err = suite.MintCoins(sdk.AccAddress(address.Bytes()), coins)
	suite.Require().NoError(err)

	// send to erc20
	err = keeper.SendCoinsFromNativeToERC20(suite.ctx, address, coins)
	suite.Require().NoError(err)

	// check erc20 balance
	contract, found := keeper.GetContractByDenom(suite.ctx, denom)
	suite.Require().True(found)

	ret, err := keeper.CallCronosERC20(suite.ctx, contract, "balanceOf", address)
	suite.Require().NoError(err)
	suite.Require().Equal(amount, big.NewInt(0).SetBytes(ret))

	ret, err = keeper.CallCronosERC20(suite.ctx, contract, "totalSupply")
	suite.Require().NoError(err)
	suite.Require().Equal(amount, big.NewInt(0).SetBytes(ret))
}
