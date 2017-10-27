const getChain = require('./getChain');

(async function () {
  const chain = await getChain();

  console.log('Install cc: ', await chain.installChaincode({
    path: 'chaincode/src/adchain',
    version: 'v0'
  }));

  console.log('Instantiate cc: ', await chain.instantiateChaincode({
    chain: 'ttl',
    path: 'chaincode/src/adchain',
    version: 'v0',
    args: ['100']
  }));

  console.log('Instantiate cc success!');

  // console.log('Read from ledger for key "abc": ');
  // console.log((await chain.queryByChaincode({
  //   name: 'adchain',
  //   fcn: 'read',
  //   args: ['abc']
  // })).map(b => b.toString()));
})();