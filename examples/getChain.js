const fabric = require('../');

// Must be Admin role: crypto-config\peerOrganizations\org1.example.com\users\Admin@org1.example.com\msp\keystore
const key =
`-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgkJ20fKXCkH62sGL5
V9l144ypEBIcpUYar0rIXOGeByGhRANCAASuwfO10R6M99UthHtneOgZ6Fc6U7cP
azUotTQklx0WzfwwuF+SGn1kkVp+Sm3CC7gZ9jXKVNs38ACetqI4z5yv
-----END PRIVATE KEY-----`;

// Must be Admin role: crypto-config\peerOrganizations\org1.example.com\users\Admin@org1.example.com\msp\signcerts
const cert =
`-----BEGIN CERTIFICATE-----
MIICLjCCAdWgAwIBAgIRAJw1zfWT+j/sW1JCAvQghkIwCgYIKoZIzj0EAwIwczEL
MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xGTAXBgNVBAoTEG9yZzEuZXhhbXBsZS5jb20xHDAaBgNVBAMTE2Nh
Lm9yZzEuZXhhbXBsZS5jb20wHhcNMTcwNjAxMTEyNzU0WhcNMjcwNTMwMTEyNzU0
WjBbMQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMN
U2FuIEZyYW5jaXNjbzEfMB0GA1UEAwwWQWRtaW5Ab3JnMS5leGFtcGxlLmNvbTBZ
MBMGByqGSM49AgEGCCqGSM49AwEHA0IABK7B87XRHoz31S2Ee2d46BnoVzpTtw9r
NSi1NCSXHRbN/DC4X5IafWSRWn5KbcILuBn2NcpU2zfwAJ62ojjPnK+jYjBgMA4G
A1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAMBgNVHRMBAf8EAjAA
MCsGA1UdIwQkMCKAIPK0VS0NhtH0vYEC5prOcc9+7N6nIRpJZFQTuUFGPTOhMAoG
CCqGSM49BAMCA0cAMEQCIBUCwpGxUXOHjuVSxbL4TSrA5N+/FK3+K9F5T7y/2/KI
AiAcZZSSbAtmt5UsrrnFQ7ET44r5bNlpuiOAXyypiviBCQ==
-----END CERTIFICATE-----`;


async function fromCert() {
  console.log('Enroll with cert.');

  return await fabric.getChain(
    {
      enrollment: {
        enrollmentID: 'test-client',
        key,
        cert
      },
      uuid:'test',
      channelId: 'ttl',
      orderer: {
        url: 'grpcs://localhost:7050',
        pemPath: 'channel/crypto-config/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt',
        sslTargetNameOverride: 'orderer.example.com'
      },
      peers: [
        {
            url: 'grpcs://localhost:7051',
            pemPath: 'channel/crypto-config/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt',
            sslTargetNameOverride: 'peer0.org1.example.com'
        }
      ],
      eventUrl: 'grpcs://localhost:7053',
      mspId: 'Org1MSP'
    }
  );
}

async function fromCa() {
  console.log('Enroll with ca server.');

  return await fabric.getChain(
    {
      enrollment: {
        enrollmentID: 'admin',
        enrollmentSecret: 'adminpw',
        ou: 'COP'
      },
      uuid:'test',
      channelId: 'ttl',
      orderer: {
          url: 'grpcs://localhost:7050',
          pemPath: 'channel/crypto-config/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt',
          sslTargetNameOverride: 'orderer.example.com'
      },
      peers: [
          {
              url: 'grpcs://localhost:7051',
              pemPath: 'channel/crypto-config/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt',
              sslTargetNameOverride: 'peer0.org1.example.com'
          }
      ],
      eventUrl: 'grpcs://localhost:7053',
      caUrl: 'http://localhost:7054',
      mspId: 'Org1MSP',
    }
  )
}

module.exports = /^true/i.test(process.env.USE_CA) ? fromCa : fromCert;