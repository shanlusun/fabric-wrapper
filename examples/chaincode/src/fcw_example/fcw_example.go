package main

import (
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
	pb_timestamp "github.com/golang/protobuf/ptypes/timestamp"
)

// SimpleChaincode example simple Chaincode implementation.
type SimpleChaincode struct {
}


// Org registering schema is used for registering a organization on chain.
// ORG_REGISTER
// To store this data the key will be: md5_hash(cert)
type org_registering struct {
	OperationType	string	`json:"operationType"` //operationType is used to distinguish the various types of operations(DATA_REGISTER)
	Owner 			string 	`json:"owner"`   //owner is the md5 hash value of cert
	OrgName     	string 	`json:"orgName"` //organization name from subject of cert
	CommonName		string 	`json:"commonName"` //common name from subject of cert
	Timestamp   	pb_timestamp.Timestamp   `json:"timestamp"` //the time when the action happens
}

// Data registering schema is used for uploading a new file.
// DATA_REGISTER
// To store this data the key will be: md5_hash(cert) + "_" + dataName
type data_registering struct {
	OperationType	string	`json:"operationType"` //operationType is used to distinguish the various types of operations(DATA_REGISTER)
	DataType 		string 	`json:"dataType"`   //dataType is used to distinguish the various types of files(the key is phone number or imei etc.)
	Owner      		string 	`json:"owner"`    //owner is the md5 hash value of cert
	DataName       	string 	`json:"dataName"`
	LineCount      	int 	`json:"lineCount"`
	Timestamp   	pb_timestamp.Timestamp   `json:"timestamp"` //the time when the action happens
}

// On boarding schema is used for matching.
// ON_BOARDING
// To store this data the key will be: TxID + "_" + Step
type on_boarding struct {
	OperationType	string	`json:"operationType"` //operationType is used to distinguish the various types of operations(ON_BOARDING)
	TxID			string  `json:"txID"`	  //txID of step 1 to track like sessionId for one matching
	Step 			int 	`json:"step"`
	Owner      		string 	`json:"owner"`    //owner is the md5 hash value of cert
	//DataType string `json:"dataType"` //dataType is used to distinguish the various types of objects in state database
	DataName       	string 	`json:"dataName"`
	FilteredLineCount	int	`json:"filteredLineCount"`  //each step the data will be filtered by Bloom, this field counts the remain lines after filtering.
	TargetOwner     string 	`json:"targetOwner"`
	TargetDataName  string  `json:"targetDataName"`
	IsFinished		bool 	`json:"isFinished"`
	BloomURI		string 	`json:"bloomURI"`
	Timestamp   	pb_timestamp.Timestamp   `json:"timestamp"` //the time when the action happens
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

	idBytes, err := getCert(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	ownerId, err := md5_hash(idBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	operationType := args[0]
	// ==== Input sanitation ====
	// ORG_REGISTER will only happen when the peer first time try to start a transaction(like uploading new file)
	// SDK client should try to do ORG_REGISTER when starts, if the ORG_REGISTER is already done before, nothing will be happen here.
	//TODO: change to switch with fewer functions
	if operationType == "ORG_REGISTER" {
		//-----only 1 parameter------
		// 		0
		// "ORG_REGISTER"

        orgName, commonName, err := getOrgNameAndCommonName(idBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
		//If the ownerId already registered before, just return the history registered info.
		queryResults, err := queryByOwnerAndOperationType(stub, operationType, ownerId)
		if err != nil {
			return shim.Error(err.Error())
		}
		if len(queryResults) > 0 {
			return shim.Success(queryResults)
		}

		// === prepare the org json ===
		txTimestamp, err := getTxTimestamp(stub)
		if err != nil {
			return shim.Error(err.Error())
		}
		data := &org_registering{operationType,ownerId,orgName,commonName, txTimestamp}
		dataJSONasBytes, err := json.Marshal(data)
		if err != nil {
			return shim.Error(err.Error())
		}

		// === Save org to state ===
		key := ownerId
		fmt.Printf("starting PutState, key:%s, value:%s\n", key, string(dataJSONasBytes))
		err = stub.PutState(key, dataJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}

	} else if operationType == "DATA_REGISTER" {
		//---------------4 parameters-------------------
		//   0       			1       	2     		3
		// "DATA_REGISTER", "DataType", "DataName", "LineCount"

		if len(args) != 4 {
			return shim.Error("Incorrect number of arguments. Expecting 4 parameters for DATA_REGISTER")
		}

		dataType := strings.ToLower(args[1])
		dataName := strings.ToLower(args[2])

		lineCount, err := strconv.Atoi(args[3])
		if err != nil {
			return shim.Error("4th argument must be a numeric string as lineCount of DATA_REGISTER.")
		}

		//If the ownerId already registered this data before, just return the history data registering info.
		queryResults, err := queryByDataAndOperationType(stub, operationType, ownerId, dataName)
		if err != nil {
			return shim.Error(err.Error())
		}
		if len(queryResults) > 0 {
			return shim.Success(queryResults)
		}

		// === prepare the org json ===
		txTimestamp, err := getTxTimestamp(stub)
		if err != nil {
			return shim.Error(err.Error())
		}

		data := &data_registering{operationType,dataType,ownerId,dataName, lineCount, txTimestamp}
		dataJSONasBytes, err := json.Marshal(data)
		if err != nil {
			return shim.Error(err.Error())
		}

		// === Save org to state ===
		key := ownerId + "_" + dataName
		fmt.Printf("starting PutState, key:%s, value:%s\n", key, string(dataJSONasBytes))
		err = stub.PutState(key, dataJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}

	} else if operationType == "ON_BOARDING" {
		//---------------8 parameters-------------------
		//      0       	1       	2     		  3  			   		4			 	5			    6			7
		// "ON_BOARDING", "Step",   "DataName", "FilteredLineCount",  "TargetOwner", "TargetDataName", "IsFinished", "Bloom"
		if len(args) != 8 {
			return shim.Error("Incorrect number of arguments. Expecting 8 parameters for ON_BOARDING")
		}

		step, err := strconv.Atoi(args[1])
		if err != nil {
			return shim.Error("2nd argument must be a numeric string as step of ON_BOARDING.")
		}
		if step < 1 {
			return shim.Error("2nd argument must be a numeric bigger than 0.")
		}

		dataName := strings.ToLower(args[2])
		filteredLineCount, err := strconv.Atoi(args[3])
		if err != nil {
			return shim.Error("4th argument must be a numeric string as filteredLineCount of ON_BOARDING.")
		}

		targetOwner := strings.ToLower(args[4])
		targetDataName := strings.ToLower(args[5])

		isFinished, err := strconv.ParseBool(args[6])
		if err != nil {
			return shim.Error("6th argument must be a boolean as isFinished of ON_BOARDING.")
		}

		bloomURI := strings.ToLower(args[7])

		//targetOwner should not be the same as owner
		if ownerId == targetOwner {
			return shim.Error("The targetOwner should not be the same as current owner.")
		}

		var txID string
		if step == 1 {
			//for step 1, get the tx_id of the transaction proposal, and this tx_id will be used as a tracking id until the matching step is finished.
			txID = stub.GetTxID()
			//for step 1, check whether the owner exists, whether TargetOwner exists
			queryResults, err := queryByOwnerAndOperationType(stub, "ORG_REGISTER", ownerId)
			if err != nil {
				return shim.Error(err.Error())
			}
			if len(queryResults) == 0 {
				return shim.Error(fmt.Sprintf("Current owner:%s has not registered yet, please do ORG_REGISTER first.", ownerId))
			}

			queryResults, err = queryByOwnerAndOperationType(stub, "ORG_REGISTER", targetOwner)
			if err != nil {
				return shim.Error(err.Error())
			}
			if len(queryResults) == 0 {
				return shim.Error(fmt.Sprintf("targetOwner:%s has not registered yet, please do ORG_REGISTER first.", targetOwner))
			}

			//for step 1, check whether the DataName exists, whether the TargetDataName exists
			queryResults, err = queryByDataAndOperationType(stub, "DATA_REGISTER", ownerId, dataName)
			if err != nil {
				return shim.Error(err.Error())
			}
			if len(queryResults) == 0 {
				return shim.Error(fmt.Sprintf("Current owner:%s doesn't have data:%s yet, please do DATA_REGISTER for this data first.", ownerId, dataName))
			}

			queryResults, err = queryByDataAndOperationType(stub, "DATA_REGISTER", targetOwner, targetDataName)
			if err != nil {
				return shim.Error(err.Error())
			}
			if len(queryResults) == 0 {
				return shim.Error(fmt.Sprintf("The targetOwner:%s doesn't have data:%s yet, please double check.", targetOwner, targetDataName))
			}

			//for step 1, need to check whether the matching for these pair of data ever happened before, if Yes, just return with notice.
			queryResult, err := queryByStepAndOperationType(stub, operationType, 1, ownerId, dataName, targetOwner, targetDataName)
			if err != nil {
				return shim.Error(err.Error())
			}
			if queryResult != nil && len(queryResult) > 0 {
				var dataJSON on_boarding
				err = json.Unmarshal(queryResult, &dataJSON)
				if err != nil {
					return shim.Error(err.Error())
				}
				return shim.Error(fmt.Sprintf("This ON_BOARDING action already finished before, txID:%s", dataJSON.TxID))
			}
		} else {
			//here means step > 1
			//check step, whether there is a (step - 1) happened before to make sure this is correct step. Also the step should not finished(isFinished==false)
			queryResult, err := queryByStepAndOperationType(stub, operationType, step - 1, ownerId, dataName, targetOwner, targetDataName)
			if err != nil {
				return shim.Error(err.Error())
			}
			if len(queryResult) == 0 || queryResult == nil {
				return shim.Error(fmt.Sprintf("Can not find the previous step:%d", step - 1))
			}

			var dataJSON on_boarding
			err = json.Unmarshal(queryResult, &dataJSON)
			if err != nil {
				return shim.Error(err.Error())
			}

			txID = dataJSON.TxID	//reuse the txID of previous step
			if dataJSON.IsFinished {
				return shim.Error(fmt.Sprintf("This ON_BOARDING action already finished before, txID:%s", txID))
			}
		}

		// === prepare the on_boarding json ===
		txTimestamp, err := getTxTimestamp(stub)
		if err != nil {
			return shim.Error(err.Error())
		}

		data := &on_boarding{operationType,
									 txID,
			                         step,
			                       ownerId,
			                    dataName,
			               filteredLineCount,
			                  targetOwner,
			               targetDataName,
								 isFinished,
							     bloomURI,
								txTimestamp}

		dataJSONasBytes, err := json.Marshal(data)
		if err != nil {
			return shim.Error(err.Error())
		}

		// === Save org to state ===
		key := txID + "_" + string(step)
		fmt.Printf("starting PutState, key:%s, value:%s\n", key, string(dataJSONasBytes))
		err = stub.PutState(key, dataJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
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

// ============================================================================================================================
// Query - query by operationType and ownerId:the md5_hash of cert.
// ============================================================================================================================
func queryByOwnerAndOperationType(stub shim.ChaincodeStubInterface, operationType string, ownerId string) ([]byte, error) {
	var err error
	fmt.Println("starting queryByOwnerAndOperationType")

	if len(operationType) == 0 {
		return nil, errors.New("Incorrect operationType. Expecting non empty type.")
	}
	if len(ownerId) != 32 {
		return nil, errors.New("Incorrect ownerId. Expecting 32 bytes of md5 hash.")
	}

	queryString := fmt.Sprintf("{\"selector\":{\"operationType\":\"%s\",\"owner\":\"%s\"}}", operationType, ownerId)
	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return nil, err
	}
	return queryResults, nil
}

// ============================================================================================================================
// Query - query by operationType, dataName and ownerId:the md5_hash of cert.
// ============================================================================================================================
func queryByDataAndOperationType(stub shim.ChaincodeStubInterface, operationType string, ownerId string, dataName string) ([]byte, error) {
	var err error
	fmt.Println("starting queryByDataAndOperationType")

	if len(operationType) == 0 {
		return nil, errors.New("Incorrect operationType. Expecting non empty type.")
	}
	if len(ownerId) != 32 {
		return nil, errors.New("Incorrect ownerId. Expecting 16 bytes of md5 hash.")
	}
	if len(dataName) == 0 {
		return nil, errors.New("Incorrect dataName. Expecting non empty dataName.")
	}

	queryString := fmt.Sprintf("{\"selector\":{\"operationType\":\"%s\",\"owner\":\"%s\",\"dataName\":\"%s\"}}",
		                               operationType, ownerId, dataName)
	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return nil, err
	}
	return queryResults, nil
}

// ============================================================================================================================
// Query - query by operationType, step, dataName and ownerId:the md5_hash of cert.
// ============================================================================================================================
func queryByStepAndOperationType(stub shim.ChaincodeStubInterface,
	                             operationType string,
                                 step int,
	                             ownerId string,
								 dataName string,
                                 targetOwner string,
                                 targetDataName string) ([]byte, error) {
	var err error
	fmt.Println("starting queryByDataAndOperationType")

	if len(operationType) == 0 {
		return nil, errors.New("Incorrect operationType. Expecting non empty type.")
	}
	if step < 1 {
		return nil, errors.New("Incorrect step. Expecting step >= 1.")
	}
	if len(ownerId) != 32 || len(targetOwner) != 32 {
		return nil, errors.New("Incorrect owner or targetOwner. Expecting 16 bytes of md5 hash.")
	}
	if len(dataName) == 0 || len(targetDataName) == 0 {
		return nil, errors.New("Incorrect dataName or targetDataName. Expecting non empty dataName and targetDataName.")
	}

	queryString := fmt.Sprintf("{\"selector\":{\"operationType\":\"%s\",\"step\":\"%d\",\"owner\":\"%s\",\"dataName\":\"%s\",\"targetOwner\":\"%s\",\"targetDataName\":\"%s\"}}",
		                       operationType, step, ownerId, dataName, targetOwner, targetDataName)
	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()
	if resultsIterator.HasNext() == false {
		return nil, nil 	//there is no record for step
	}
	queryResponse, err := resultsIterator.Next()
	if err != nil {
		return nil, err
	}

	return queryResponse.Value, nil
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
func md5_hash(idBytes []byte) (string, error) {
	if nil == idBytes|| len(idBytes) == 0 {
		return "", errors.New("md5_hash: input parameter idBytes is invalid.")
	}
	digest := md5.New()
	digest.Write(idBytes)
	hash_cert := digest.Sum(nil)
	fmt.Printf("MD5 Hash of idBytes Hex:%x\n", hash_cert) // 16 bytes
	return fmt.Sprintf("%x", hash_cert), nil
}

// ========================================================
// getCert is used to unmarshal the creator
// return cert []byte
// ========================================================
func getCert(stub shim.ChaincodeStubInterface) ([]byte, error) {
	creator, err := stub.GetCreator()
	if err != nil {
		fmt.Errorf("Failed to get creator info")
		return nil, err
	}

	serializedIdentity := &pb_msp.SerializedIdentity{}
	err = proto.Unmarshal(creator, serializedIdentity)
	if err != nil {
		fmt.Sprintf("Failed to Unmarshal serializedIdentity, err %s", err)
		return nil, err
	}
	return serializedIdentity.IdBytes, nil
}

func getTxTimestamp(stub shim.ChaincodeStubInterface) (pb_timestamp.Timestamp, error) {
	pTxTimestamp, err := stub.GetTxTimestamp()
	if err != nil {
		fmt.Errorf("Failed to call stub.GetTxTimestamp, err:%s\n",err.Error())
		return pb_timestamp.Timestamp{}, err
	}
	txTimestamp := pb_timestamp.Timestamp{}
	txTimestamp.Seconds = (*pTxTimestamp).Seconds
	txTimestamp.Nanos = (*pTxTimestamp).Nanos
	return txTimestamp, nil
}

// ========================================================
// Parse the cert to fetch org name and common name
// return both orgName and commonName
// ========================================================
func getOrgNameAndCommonName(idBytes []byte) (string, string, error) {
	block, _ := pem.Decode([]byte(idBytes))
	if block == nil {
		return "", "", errors.New("Failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", "", errors.New(fmt.Sprintf("Failed to ParseCertificate, err %s", err))
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

	if orgName == "" && commonName == "" {
		return "", "", errors.New("Both orgName amd commonName are empty.")
	}
	return orgName, commonName, nil
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
