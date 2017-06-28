package main

import (
	"time"
	"strings"
	"encoding/json"
	"crypto/x509"
	"encoding/pem"
	"crypto/md5"
    "bytes"
	"errors"
	"fmt"
	"strconv"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	//"github.com/hyperledger/fabric/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	//pb_common "github.com/hyperledger/fabric/protos/common"
	pb_msp "github.com/hyperledger/fabric/protos/msp"
)

// SimpleChaincode example simple Chaincode implementation.
type SimpleChaincode struct {
}


// Org registering schema is used for registering a organization on chain.
// ORG_REGISTER
type org_registering struct {
	OperationType	string	`json:"operationType"` //operationType is used to distinguish the various types of operations(DATA_REGISTER)
	Owner 			string 	`json:"owner"`   //owner is the md5 hash value of cert
	OrgName     	string 	`json:"orgName"` //organization name from subject of cert
	CommonName		string 	`json:"commonName"` //common name from subject of cert
	Timestamp   	time.Time   `json:"timestamp"` //the time when the action happens
}

// registering schema is used for uploading a new file. 
// DATA_REGISTER
type data_registering struct {
	OperationType	string	`json:"operationType"` //operationType is used to distinguish the various types of operations(DATA_REGISTER)
	DataType 		string 	`json:"dataType"`   //dataType is used to distinguish the various types of files(the key is phone number or imei etc.)
	Owner      		string 	`json:"owner"`    //owner is the md5 hash value of cert
	DataName       	string 	`json:"dataName"`
	LineCount      	int 	`json:"lineCount"`
	Timestamp   	time.Time   `json:"timestamp"` //the time when the action happens
}

// onboarding schema is used for matching.
// ON_BOARDING
type on_boarding struct {
	OperationType	string	`json:"operationType"` //operationType is used to distinguish the various types of operations(ON_BOARDING)
	TxID			string  `json:"txID"`
	Step 			int 	`json:"step"`
	Owner      		string 	`json:"owner"`    //owner is the md5 hash value of cert
	//DataType string `json:"dataType"` //dataType is used to distinguish the various types of objects in state database
	DataName       	string 	`json:"dataName"`
	TargetOwner     string 	`json:"targetOwner"`
	TargetDataName  string  `json:"targetDataName"`
	Bloom			[]byte 	`json:"bloom"`
	Timestamp   	time.Time   `json:"timestamp"` //the time when the action happens
}


// ============================================================================================================================
// Invoke - Our entry point for Invocations
// ============================================================================================================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println(" ")
	fmt.Println("starting invoke, for - " + function)

	// Handle different functions
	if function == "write" {           //generic writes to ledger
		return t.write(stub, args)
	} else if function == "read" {            //generic read ledger
		return t.read(stub, args)
	} else if function == "query" {           //query ledger with complex JSON query string
        return t.query(stub, args)
    } else if function == "submit" {           //submit new uploaded file info
		return t.submit(stub, args)
	}

	// error out
	fmt.Println("Received unknown invoke function name - " + function)
	return shim.Error("Received unknown invoke function name - '" + function + "'")
}

// ============================================================================================================================
// Init - reset all the things
// ============================================================================================================================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	_, args := stub.GetFunctionAndParameters()
	var Aval int
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	// Initialize the chaincode
	Aval, err = strconv.Atoi(args[0])
	if err != nil {
		return shim.Error("Expecting integer value for asset holding")
	}

	// Write the state to the ledger
	err = stub.PutState("abc", []byte(strconv.Itoa(Aval))) //making a test var "abc", I find it handy to read/write to it right away to test the network
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println(" - ready for action")
	return shim.Success(nil)
}

// ============================================================================================================================
// Submit - write the onboarding data into ledger
// ============================================================================================================================
func (t *SimpleChaincode) submit(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// var err error
	//_, args := stub.GetFunctionAndParameters()

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting at least 1 parameter.")
	}

	// if there is any empty string parameters, return err.
	for i := 0; i < len(args); i++ {
		if len(args[i]) <= 0 {
			return shim.Error(strconv.Itoa(i) + "th argument must be a non-empty string")
		}
	}

	idBytes := getCert(stub)
	if idBytes == nil {
		return shim.Error("Failed to call getCert.")
	}

	key := md5_hash(idBytes)
	if key == "" {
		return shim.Error("Failed to call md5_hash.")
	}

	operationType := args[0]
	// ==== Input sanitation ====
	// ORG_REGISTER will only happen when the peer first time try to start a transaction(like uploading new file)
	// SDK client should check whether ORG_REGISTER has been done or not.
	if operationType == "ORG_REGISTER" {
		//-----only 1 parameter------
		// 		0
		// "ORG_REGISTER"

        orgName, commonName := getOrgNameAndCommonName(idBytes)
		if orgName == "" && commonName == "" {
			return shim.Error("Both orgName and commonName are empty.")
		}
		//TODO: if the org name exists, notice to rename with new name.
		// === prepare the org json ===
		org := &org_registering{operationType,key,orgName,commonName, time.Now()}
		orgJSONasBytes, err := json.Marshal(org)
		if err != nil {
			return shim.Error(err.Error())
		}

		// === Save org to state ===
		fmt.Printf("starting PutState, key:%s, value:%s\n", key, string(orgJSONasBytes))
		err = stub.PutState(key, orgJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}

	} else if operationType == "DATA_REGISTER" {
		//---------------4 parameters needed-------------------
		//   0       			1       	2     		3
		// "DATA_REGISTER", "DataType", "DataName", "LineCount"

		if len(args) != 4 {
			return shim.Error("Incorrect number of arguments. Expecting 4 for DATA_REGISTER")
		}

		dataType := strings.ToLower(args[1])
		dataName := strings.ToLower(args[2])
		//TODO:if the data name exist, return err to notice with new name
		lineCount, err := strconv.Atoi(args[3])
		if err != nil {
			return shim.Error("4th argument must be a numeric string as lineCount of DATA_REGISTER.")
		}

		// === prepare the org json ===
		data := &data_registering{operationType,dataType,key,dataName, lineCount, time.Now()}
		dataJSONasBytes, err := json.Marshal(data)
		if err != nil {
			return shim.Error(err.Error())
		}

		// === Save org to state ===
		fmt.Printf("starting PutState, key:%s, value:%s\n", key, string(dataJSONasBytes))
		err = stub.PutState(key, dataJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}

	} else if operationType == "ON_BOARDING" {
		//   0       		1       	2     		  3  			   4			 5
		// "ON_BOARDING", "Step",   "DataName",  "TargetOwner", "TargetDataName", "Bloom"

	}


	return shim.Success(nil)
}

// ============================================================================================================================
// Write - genric write variable into ledger
// ============================================================================================================================
func (t *SimpleChaincode) write(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, value string                           // Entities
	var err error
	fmt.Println("starting write")

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2. name of the variable and value to set")
	}

	name = args[0]                                   //rename for funsies
	value = args[1]
	err = stub.PutState(name, []byte(value))         //write the variable into the ledger
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end write")
	return shim.Success(nil)
}

// ============================================================================================================================
// Read - read a generic variable from ledger
// ============================================================================================================================
func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error
	fmt.Println("starting read")

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the var to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetState(name)           //get the var from ledger
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return shim.Error(jsonResp)
	}

	fmt.Println("- end read")
	return shim.Success(valAsbytes)                  //send it onward
}

// ============================================================================================================================
// Query - query a generic variable from ledger with complex query string in JSON format.
// ============================================================================================================================
func (t *SimpleChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	fmt.Println("starting query")

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting query string of JSON to query")
	}

	queryString := args[0]
    queryResults, err := getQueryResultForQueryString(stub, queryString)
    if err != nil {
        return shim.Error(err.Error())
    }
    return shim.Success(queryResults)
}

// =========================================================================================
// getQueryResultForQueryString executes the passed in query string.
// Result set is built and returned as a byte array containing the JSON results.
// =========================================================================================
func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {

	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)

	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryRecords
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getQueryResultForQueryString queryResult:\n%s\n", buffer.String())

	return buffer.Bytes(), nil
}

// ========================================================
// Input Sanitation - dumb input checking, look for empty strings
// ========================================================
func sanitize_arguments(strs []string) error{
	for i, val:= range strs {
		if len(val) <= 0 {
			return errors.New("Argument " + strconv.Itoa(i) + " must be a non-empty string")
		}
		if len(val) > 32 {
			return errors.New("Argument " + strconv.Itoa(i) + " must be <= 32 characters")
		}
	}
	return nil
}

// ========================================================
// md5_hash is used to calculate md5 hash
// return 16 bytes md5 hash
// ========================================================
func md5_hash(idBytes []byte) string {
	if nil == idBytes|| len(idBytes) == 0 {
		fmt.Println("md5_hash: input parameter idBytes is invalid.")
		return ""
	}
	digest := md5.New()
	digest.Write(idBytes)
	hash_cert := digest.Sum(nil)
	fmt.Printf("MD5 Hash of idBytes Hex:%x\n", hash_cert) // 16 bytes
	return fmt.Sprintf("%x", hash_cert)
}

// ========================================================
// getCert is used to unmarshal the creator
// return cert []byte
// ========================================================
func getCert(stub shim.ChaincodeStubInterface) []byte {
	creator, err := stub.GetCreator()
	if err != nil {
		fmt.Errorf("Failed to get creator info")
		return nil
	}

	serializedIdentity := &pb_msp.SerializedIdentity{}
	err = proto.Unmarshal(creator, serializedIdentity)
	if err != nil {
		fmt.Sprintf("Failed to Unmarshal serializedIdentity, err %s", err)
		return nil
	}
	return serializedIdentity.IdBytes
}

// ========================================================
// Parse the cert to fetch org name and common name
// return both orgName and commonName
// ========================================================
func getOrgNameAndCommonName(idBytes []byte) (string, string) {
	block, _ := pem.Decode([]byte(idBytes))
	if block == nil {
		fmt.Errorf("Failed to parse certificate PEM")
		return "", ""
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		fmt.Errorf("Failed to ParseCertificate, err %s", err)
		return "", ""
	}

	orgNameArray := cert.Subject.Organization
	var orgName string
	if len(orgNameArray) == 0 {
		orgName = ""
	} else {
		orgName = orgNameArray[0]
	}
	fmt.Printf("orgName:%s\n", orgName)

	commonName := cert.Subject.CommonName
	fmt.Printf("commonName:%s\n", commonName)

	return orgName, commonName
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode - %s", err)
	}
}
