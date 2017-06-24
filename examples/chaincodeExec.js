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
  const executeResult = Promise.all([
    chain.invokeChaincode({
      name: 'fcw_example',
      fcn: 'write',
      args: ['ab', JSON.stringify({ a: 2 })]
    }),
    chain.invokeChaincode({
      name: 'fcw_example',
      fcn: 'write',
      args: ['bc', JSON.stringify({ a: { '$inc': 100 }, b:"hello"})]
    })
  ]);
  console.log(await executeResult);

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
      fcn: 'query',
      args: [JSON.stringify({ selector:{ a: 2 }})]
  })).map(b => b.toString()));

})();