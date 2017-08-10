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

  console.log('Write to ledger for key "ab" & "bc": ');
    await chain.invokeChaincode({
      name: 'fcw_example',
      fcn: 'write',
      args: ['ab', JSON.stringify({ a: 2 })]
    })

    await chain.invokeChaincode({
      name: 'fcw_example',
      fcn: 'write',
      args: ['bc', JSON.stringify({ a: { '$inc': 100 }, b:"hello"})]
    })

    await chain.invokeChaincode({
        name: 'fcw_example',
        fcn: 'OrgRegister',
        args: []
    })

    await chain.invokeChaincode({
        name: 'fcw_example',
        fcn: 'DataRegister',
        args: ["imei", "TestFileName", "100", "", ""]
    })
    await chain.invokeChaincode({
        name: 'fcw_example',
        fcn: 'DataRegister',
        args: ["imei", "TestFileName2", "100", "", ""]
    }),
    await chain.invokeChaincode({
        name: 'fcw_example',
        fcn: 'OnBoarding',
        args: ["4", "eccd405a6833518aea9b27f7b4be78b0", "TestFileName", "10", "eccd405a6833518aea9b27f7b4be78b0", "TestFileName2", "true", "TestURI"]
    })


  console.log('Read from ledger for key "ab": ');
  console.log((await chain.queryByChaincode({
    name: 'fcw_example',
    fcn: 'read',
    args: ['ab']
  })).map(b => b.toString()));

  console.log('Read from ledger for key "bc": ');
  console.log((await chain.queryByChaincode({
    name: 'fcw_example',
    fcn: 'read',
    args: ['bc']
  })).map(b => b.toString()));

  console.log('Query from ledger: ');
  console.log((await chain.queryByChaincode({
      name: 'fcw_example',
      fcn: 'Query',
      args: [JSON.stringify({ selector:{ a: 2 }})]
  })).map(b => b.toString()));


  console.log('Test for Query(): ');
    console.log((await chain.queryByChaincode({
        name: 'fcw_example',
        fcn: 'Query',
        args: [JSON.stringify({ selector:{ owner: "eccd405a6833518aea9b27f7b4be78b0" }})]
    })).map(b => b.toString()));

    console.log('Test for WhoAmI(): ');
    console.log((await chain.queryByChaincode({
        name: 'fcw_example',
        fcn: 'WhoAmI',
        args: []
    })).map(b => b.toString()));

})();