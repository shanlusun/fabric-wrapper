const getChain = require('./getChain');

(async function () {
  const chain = await getChain();

  console.log('Query installed cc for peer0: ', await chain.queryInstalledChaincodes(0));
  console.log('Query instantiated cc: ', await chain.queryInstantiatedChaincodes());

  chain.eventhub.registerBlockEvent(block => {
    const decoded = chain.extractCcExecInfo(block);
    console.log('Cc executed: \n', decoded);
    console.log(decoded.payloads);
  });

  console.log('Write to ledger: ');

    await chain.invokeChaincode({
        name: 'adchain',
        fcn: 'OrgRegister',
        args: []
    })

    await chain.invokeChaincode({
        name: 'adchain',
        fcn: 'DataRegister',
        args: ["imei", "TestFileName", "100", "", ""]
    })
    await chain.invokeChaincode({
        name: 'adchain',
        fcn: 'DataRegister',
        args: ["imei", "TestFileName2", "100", "", ""]
    }),
    await chain.invokeChaincode({
        name: 'adchain',
        fcn: 'OnBoarding',
        args: ["1", "eccd405a6833518aea9b27f7b4be78b0", "TestFileName", "10", "eccd405a6833518aea9b27f7b4be78b0", "TestFileName2", "true", "TestURI"]
    })

  //new added for paneling
  await chain.invokeChaincode({
    name: 'adchain',
    fcn: 'DataRegister',
    args: ["imei", "TestFileName3", "100", "", "", "gender", "male"]
  }),
  await chain.invokeChaincode({
    name: 'adchain',
    fcn: 'PanelRequest',
    args: ["imei", "TestFileName", "eccd405a6833518aea9b27f7b4be78b0|eccd405a6833518aea9b27f7b4be78b0", "gender"]
  })

  // await chain.invokeChaincode({
  //   name: 'adchain',
  //   fcn: 'PanelUpdate',
  //   args: ["ad1a966ddf64d2b5d2962a7b342e0b68001c5c482cbc4ad739171c60189ae520", "true", "gender|male|10|testHLL_URI_1", "gender|female|11|testHLL_URI_2", "gender|all|12|testHLL_URI_3"]
  // })

  console.log('Query from ledger: ');

  console.log('Test for Query(): ');
    console.log((await chain.queryByChaincode({
        name: 'adchain',
        fcn: 'Query',
        args: [JSON.stringify({ selector:{ owner: "eccd405a6833518aea9b27f7b4be78b0" }})]
    })).map(b => b.toString()));

    console.log('Test for WhoAmI(): ');
    console.log((await chain.queryByChaincode({
        name: 'adchain',
        fcn: 'WhoAmI',
        args: []
    })).map(b => b.toString()));

  //new added for paneling


})();