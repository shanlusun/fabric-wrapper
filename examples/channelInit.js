const getChain = require('./getChain');
const fs = require('fs');

(async function () {
  const chain = await getChain();

  const tx = fs.readFileSync(__dirname + '/docker-compose-couch/channel-artifacts/channel_ttl.tx');
  console.log('Create channel: ', await chain.createChannel('ttl', tx));

  await new Promise(resolve => setTimeout(resolve, 6000));

  console.log('Join channel: ', await chain.joinChannel());
})();