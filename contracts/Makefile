SOLC_VERSION := 0.6.11
all    :; dapp --use solc:${SOLC_VERSION} build
clean  :; dapp clean
test   :; dapp --use solc:${SOLC_VERSION} test
deploy :; dapp create CronosERC20
