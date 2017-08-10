const fabric = require('../');

// Must be Admin role: crypto-config\peerOrganizations\org1.example.com\users\Admin@org1.example.com\msp\keystore
const key =
    `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgXYaz52yfoJPqEQ/q
zyqsNoCLM9VVAIS0o8Y4McusOAGhRANCAAQgnwSZp6Jzf+ZqNW8t5TGBjeS376hs
eIMyJLmCz06NNUGAfKRYUS7I/Zlt+6qp6XmViq/IML3TzF2YZduhkxPl
-----END PRIVATE KEY-----`;

// Must be Admin role: crypto-config\peerOrganizations\org1.example.com\users\Admin@org1.example.com\msp\signcerts
const cert =
    `-----BEGIN CERTIFICATE-----
MIICGjCCAcCgAwIBAgIRANZxWqQVaur1UqzSNCnKbG4wCgYIKoZIzj0EAwIwczEL
MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xGTAXBgNVBAoTEG9yZzEuZXhhbXBsZS5jb20xHDAaBgNVBAMTE2Nh
Lm9yZzEuZXhhbXBsZS5jb20wHhcNMTcwNjI3MDkzNjA3WhcNMjcwNjI1MDkzNjA3
WjBbMQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMN
U2FuIEZyYW5jaXNjbzEfMB0GA1UEAwwWQWRtaW5Ab3JnMS5leGFtcGxlLmNvbTBZ
MBMGByqGSM49AgEGCCqGSM49AwEHA0IABCCfBJmnonN/5mo1by3lMYGN5LfvqGx4
gzIkuYLPTo01QYB8pFhRLsj9mW37qqnpeZWKr8gwvdPMXZhl26GTE+WjTTBLMA4G
A1UdDwEB/wQEAwIHgDAMBgNVHRMBAf8EAjAAMCsGA1UdIwQkMCKAIEwoyRFk1VCK
csDR18OKwuOL1pJTR2g+kB+PP+8hiaFWMAoGCCqGSM49BAMCA0gAMEUCIQDYcagQ
hEZyHvK7VLsxM5/d5CB9FS+ScRaNEMT0ffa/lAIgfRXOg8iWjciepXKvekAqrCJn
5+mzHbMTX46Skaw4ibU=
-----END CERTIFICATE-----`;


// process.env.USE_TLS = true;
const protocol = process.env.USE_TLS ? 'grpcs' : 'grpc';

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
        url: `${protocol}://localhost:7050`,
        pemPath: process.env.USE_TLS && "docker-compose-couch/crypto-config/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt",
        sslTargetNameOverride: 'orderer.example.com'
      },
      peers: [
        {
          url: `${protocol}://localhost:7051`,
          eventUrl: `${protocol}://localhost:7053`,
          pemPath: process.env.USE_TLS && "docker-compose-couch/crypto-config/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt",
          sslTargetNameOverride: 'peer0.org1.example.com'
        },
          {
              url: `${protocol}://localhost:8051`,
              pemPath: process.env.USE_TLS && "docker-compose-couch/crypto-config/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls/ca.crt",
              sslTargetNameOverride: 'peer1.org1.example.com'
          }
      ],
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
            url: `${protocol}://localhost:7050`,
            pemPath: process.env.USE_TLS && "docker-compose-couch/crypto-config/ordererOrganizations/example.com/orderers/orderer.example.com/tls/ca.crt",
            sslTargetNameOverride: 'orderer.example.com'
        },
        peers: [
            {
                url: `${protocol}://localhost:7051`,
                eventUrl: `${protocol}://localhost:7053`,
                pemPath: process.env.USE_TLS && "examples/docker-compose-couch/crypto-config/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt",
                sslTargetNameOverride: 'peer0.org1.example.com'
            },
            {
                url: `${protocol}://localhost:8051`,
                pemPath: process.env.USE_TLS && "examples/docker-compose-couch/crypto-config/peerOrganizations/org1.example.com/peers/peer1.org1.example.com/tls/ca.crt",
                sslTargetNameOverride: 'peer1.org1.example.com'
            }
        ],
      caUrl: 'http://localhost:7054',
      mspId: 'Org1MSP'
    }
  )
}

module.exports = /^true/i.test(process.env.USE_CA) ? fromCa : fromCert;