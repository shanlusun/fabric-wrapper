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
	pb "github.com/hyperledger/fabric/protos/peer"
	pb_msp "github.com/hyperledger/fabric/protos/msp"
	pb_timestamp "github.com/golang/protobuf/ptypes/timestamp"
)


// Org registering schema is used for registering a organization on chain.
// To store this data the key will be: md5_hash(cert)
type OrgRegistering struct {
	OperationType	string	`json:"operationType"` //operationType is used to distinguish the various types of operations(DataRegister)
	Owner 			string 	`json:"owner"`   //owner is the md5 hash value of cert
	OrgName     	string 	`json:"orgName"` //organization name from subject of cert
	CommonName		string 	`json:"commonName"` //common name from subject of cert
	Timestamp   	pb_timestamp.Timestamp   `json:"timestamp"` //the time when the action happens
}

// Data registering schema is used for uploading a new file.
// To store this data the key will be: md5_hash(cert) + "_" + dataName
type DataRegistering struct {
	OperationType	string	`json:"operationType"` //operationType is used to distinguish the various types of operations(DataRegister)
	DataType 		string 	`json:"dataType"`   //dataType is used to distinguish the various types of files(the key is phone number or imei etc.)
	Owner      		string 	`json:"owner"`    //owner is the md5 hash value of cert
	DataName       	string 	`json:"dataName"`
	LineCount      	int 	`json:"lineCount"`
	HLL				string	`json:"hll"`	//not used for now
	Bloom			string	`json:"bloom"`	//not used for now
	Tag				string  `json:"tag"`	//necessary for panel
	Field 			string  `json:"field"`	//necessary for panel
	Timestamp   	pb_timestamp.Timestamp   `json:"timestamp"` //the time when the action happens
	MatchCount		int		`json:"matchCount"`		//how many times the data has ever been matched before.
	LastMatchTimestamp	pb_timestamp.Timestamp   `json:"lastMatchTimestamp"` //the time when the data participated matching before.
}

// On boarding schema is used for matching.
// To store this data the key will be: TxID + "_" + Step
type OnBoarding struct {
	OperationType	string	`json:"operationType"` //operationType is used to distinguish the various types of operations(OnBoarding)
	TxID			string  `json:"txID"`	  //txID of step 1 to track like sessionId for one matching
	Step 			int 	`json:"step"`
	Owner      		string 	`json:"owner"`    //owner is the md5 hash value of cert
	//DataType string `json:"dataType"` //dataType is used to distinguish the various types of objects in state database
	DataName       	string 	`json:"dataName"`
	FilteredLineCount	int	`json:"filteredLineCount"`  //each step the data will be filtered by Bloom, this field counts the remain lines after filtering.
	TargetOwner     string 	`json:"targetOwner"`
	TargetDataName  string  `json:"targetDataName"`
	IsFinished		bool 	`json:"isFinished"`
	//BloomURI		string 	`json:"bloomURI"`
	Timestamp   	pb_timestamp.Timestamp   `json:"timestamp"` //the time when the action happens
}

type QueryResult_DataRegistering struct {
	Key 	string 	`json:"Key"`
	Record	DataRegistering 	`json:"Record"`
}

type QueryResult_DataRegistering_Array []*QueryResult_DataRegistering

// New schemas used for panel
type Paneling struct {
	OperationType	string	`json:"operationType"` //operationType is used to distinguish the various types of operations(Paneling)
	TxID			string  `json:"txID"`	  //txID is used to tracking all the progress of panel
	Sponsor  		string	`json:"sponsor"`    //sponsor is the ownerId which is the md5 hash value of cert
	DataType 		string 	`json:"dataType"`   //dataType is used to distinguish the various types of files(the key is phone number or imei etc.)
	DataName		string 	`json:"dataName"`
	Providers      	Providers 	`json:"providers"`
	IsFinished		bool 	`json:"isFinished"`
	Timestamp   	pb_timestamp.Timestamp   `json:"timestamp"` //the time when the action happens
	LastUpdatedTimestamp	pb_timestamp.Timestamp   `json:"lastUpdatedTimestamp"` //the time when the data updated.
}

type Providers struct {
	GenderProviderArray	[]GenderProvider	`json:"genderProviderArray"`
}

type GenderProvider struct {
	ProviderId		string	`json:"providerId"`    //owner is the md5 hash value of cert
	Gender			Gender	`json:"gender"`
}

type Gender struct {
	Male	DataDigest	`json:"male"`
	Female	DataDigest	`json:"female"`
	All     DataDigest	`json:"all"`
}

type DataDigest struct {
	LineCount      	int 	`json:"lineCount"`
	HLL				string	`json:"hll"`
	//Bloom			string	`json:"bloom"`  //not used for now
}


type AdChainChaincode struct {
}

// Init initialization
func (t *AdChainChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (t *AdChainChaincode) Put(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	var key = string(args[0])
	var val = args[1]

	stub.PutState(key, val)
	return shim.Success(nil)
}

// Invoke runs callback representing the invocation of a chaincode
func (t *AdChainChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, _ := stub.GetFunctionAndParameters()
	fmt.Println(" ")
	fmt.Println("starting invoke, for - " + function)

	// Handle different functions
	if function == "Query" {
		return t.Query(stub)
	} else if function == "OrgRegister" {
		return t.OrgRegister(stub)
	} else if function == "DataRegister" {
		return t.DataRegister(stub)
	} else if function == "OnBoarding" {
		return t.OnBoarding(stub)
	} else if function == "WhoAmI" {
		return t.WhoAmI(stub)
	} else if function == "PanelRequest" {
		return t.PanelRequest(stub)
	} else if function == "PanelUpdate" {
		return t.PanelUpdate(stub)
	}

	return shim.Error("Received unknown function invocation")
}

// ============================================================================================================================
// OrgRegister will only happen when the peer first time try to start a transaction(like uploading new file)
// SDK client should try to do OrgRegister when starts, if the OrgRegister is already done before, nothing will be happen here.
// ============================================================================================================================
func (t *AdChainChaincode) OrgRegister(stub shim.ChaincodeStubInterface) pb.Response {
	function, _ := stub.GetFunctionAndParameters()

	operationType := function
	emptyQueryResults := []byte("[]")

	ownerId, err := generateOwnerIdByCert(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	orgName, commonName, err := getOrgNameAndCommonName(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	//If the ownerId already registered before, just return.
	queryResults, err := queryByOwnerAndOperationType(stub, operationType, ownerId)
	if err != nil {
		return shim.Error(err.Error())
	}
	if bytes.Equal(queryResults[:], emptyQueryResults[:]) == false {
		fmt.Printf("Already did OrgRegister:%s\n", queryResults)
		return shim.Success(nil)
	}

	// === prepare the org json ===
	txTimestamp, err := getTxTimestamp(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	data := &OrgRegistering{operationType,ownerId,orgName,commonName, txTimestamp}
	dataJSONasBytes, err := json.Marshal(data)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === Save org to state ===
	key := operationType + "_" + ownerId
	fmt.Printf("Starting PutState, key:%s, value:%s\n", key, string(dataJSONasBytes))
	err = stub.PutState(key, dataJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// ============================================================================================================================
// DataRegister will only happen when the peer first time try to start a transaction(like uploading new file)
// If the DataRegister is already done before, nothing will happen here.
// ============================================================================================================================
func (t *AdChainChaincode) DataRegister(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	//-------------3 parameters is necessary------------
	//     0       		1       	2		3		  4			5						6
	// "DataType", "DataName", "LineCount" "HLL"	"Bloom"	  "Tag"(optional)	  "Field"(optional)

	// ==== Input sanitation ====
	if len(args) < 5 {
		return shim.Error("Incorrect number of arguments. Expecting at least 5 parameters for DataRegister")
	}
	//if there is any empty string parameters, return err.
	//does not check for last 4 arguments
	for i := 0; i < 3; i++ {
		if len(args[i]) <= 0 {
			return shim.Error(strconv.Itoa(i) + "th argument must be a non-empty string")
		}
	}

	ownerId, err := generateOwnerIdByCert(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	operationType := function
	emptyQueryResults := []byte("[]")

	dataType := strings.ToLower(args[0])
	dataName := args[1]

	lineCount, err := strconv.Atoi(args[2])
	if err != nil {
		return shim.Error("3th argument must be a numeric string as lineCount of DataRegister.")
	}

	hll := args[3]
	bloom := args[4]

	//necessary for panel, currently only have one tag:gender, field might be one of: male; female; all
	var tag string
	var field string
	if len(args) >= 7 {
		tag = args[5]
		field = args[6]
	}

	//If the ownerId already registered this data before, just return.
	queryResults, err := queryByDataAndOperationType(stub, operationType, ownerId, dataName)
	if err != nil {
		return shim.Error(err.Error())
	}
	if bytes.Equal(queryResults[:], emptyQueryResults[:]) == false {
		fmt.Printf("Already did DataRegister:%s\n", queryResults)
		return shim.Success(nil)
	}

	// === prepare the org json ===
	txTimestamp, err := getTxTimestamp(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	data := &DataRegistering{operationType,
							 dataType,
							 ownerId,
							 dataName,
							 lineCount,
							 hll,
							 bloom,
		                     tag,
		                     field,
							 txTimestamp,
							 0,
							 pb_timestamp.Timestamp{0,0}} // lastMatchTimestamp is 0 when registering.

	dataJSONasBytes, err := json.Marshal(data)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === Save data to state ===
	key := operationType + "_" + ownerId + "_" + dataName
	fmt.Printf("Starting PutState, key:%s, value:%s\n", key, string(dataJSONasBytes))
	err = stub.PutState(key, dataJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

// ====================================================================================================================================
// OnBoarding is the main function used to start matching data between owner and targetOwner.
// The step starts from '1', and SDK clients will listen on event whether targetOwner is the same as theirs, if 'Yes' starts OnBoarding
// =====================================================================================================================================
func (t *AdChainChaincode) OnBoarding(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	//---------------------------------------7 parameters-------------------------------------------------
	//     0       	 1       		2     		  		3  			   	  4			 	  5			  6				7		  8
	//  "Step",   "OwnerId",	"DataName", "FilteredLineCount",  "TargetOwner", "TargetDataName", "IsFinished", "Bloom"	"TxID"(optional)

	//TODO: TxID is added for panel to track all the progress for panel transaction(PanelRequest, OnBoarding, ... etc.)
	//TODO: Add checking for dataType of both data, should be the same

	// ==== Input sanitation ====
	if len(args) < 8 {
		return shim.Error("Incorrect number of arguments. Expecting at least 8 parameters for OnBoarding")
	}
	// if there is any empty string parameters, return err.
	for i := 0; i < 8; i++ {
		if len(args[i]) <= 0 {
			return shim.Error(strconv.Itoa(i) + "th argument must be a non-empty string")
		}
	}

	operationType := function
	emptyQueryResults := []byte("[]")

	step, err := strconv.Atoi(args[0])
	if err != nil {
		return shim.Error("1st argument must be a numeric string as step of OnBoarding.")
	}
	if step < 1 {
		return shim.Error("1st argument must be a numeric bigger than 0.")
	}

	ownerId := strings.ToLower(args[1])
	dataName := args[2]
	filteredLineCount, err := strconv.Atoi(args[3])
	if err != nil {
		return shim.Error("4rd argument must be a numeric string as filteredLineCount of OnBoarding.")
	}

	targetOwner := strings.ToLower(args[4])
	targetDataName := args[5]

	isFinished, err := strconv.ParseBool(args[6])
	if err != nil {
		return shim.Error("6th argument must be a boolean as isFinished of OnBoarding.")
	}

	//for step 1, get the tx_id of the transaction proposal, and this tx_id will be used as a tracking id until the matching step is finished.
	var txID string = stub.GetTxID()
	//if the TxID is passed from argument which means we need to reuse that TxID as desired, currently this only happens for paneling.
	if len(args) > 8 && len(args[8]) > 0 {
		txID = args[8]
	}

	////targetOwner should not be the same as owner
	//if ownerId == targetOwner {
	//	return shim.Error("The targetOwner should not be the same as current owner.")
	//}

	if step == 1 {
		//for step 1, check whether the owner is current owner
		currentOwnerId, err := generateOwnerIdByCert(stub)
		if err != nil {
			return shim.Error(err.Error())
		}
        if currentOwnerId != ownerId {
			return shim.Error(fmt.Sprintf("Current ownerId:%s does not equal to the ownerId:%s in argument, step=1", currentOwnerId, ownerId))
		}

		//for step 1, check whether the owner exists, whether TargetOwner exists
		queryResults, err := queryByOwnerAndOperationType(stub, "OrgRegister", ownerId)
		if err != nil {
			return shim.Error(err.Error())
		}
		if bytes.Equal(queryResults[:], emptyQueryResults[:]) {
			fmt.Sprintf("Current owner:%s has not registered yet, please do OrgRegister first.\n", ownerId)
			return shim.Error(fmt.Sprintf("Current owner:%s has not registered yet, please do OrgRegister first.", ownerId))
		}

		queryResults, err = queryByOwnerAndOperationType(stub, "OrgRegister", targetOwner)
		if err != nil {
			return shim.Error(err.Error())
		}
		if bytes.Equal(queryResults[:], emptyQueryResults[:]) {
			fmt.Sprintf("targetOwner:%s has not registered yet, please do OrgRegister first.\n", targetOwner)
			return shim.Error(fmt.Sprintf("targetOwner:%s has not registered yet, please do OrgRegister first.", targetOwner))
		}

		//for step 1, check whether the DataName exists, whether the TargetDataName exists
		queryResults, err = queryByDataAndOperationType(stub, "DataRegister", ownerId, dataName)
		if err != nil {
			return shim.Error(err.Error())
		}
		if bytes.Equal(queryResults[:], emptyQueryResults[:]) {
			fmt.Sprintf("Current owner:%s doesn't have data:%s yet, please do DataRegister for this data first.\n", ownerId, dataName)
			return shim.Error(fmt.Sprintf("Current owner:%s doesn't have data:%s yet, please do DataRegister for this data first.", ownerId, dataName))
		}

		queryResults, err = queryByDataAndOperationType(stub, "DataRegister", targetOwner, targetDataName)
		if err != nil {
			return shim.Error(err.Error())
		}

		if bytes.Equal(queryResults[:], emptyQueryResults[:]) {
			fmt.Sprintf("The targetOwner:%s doesn't have data:%s yet, please double check.\n", targetOwner, targetDataName)
			return shim.Error(fmt.Sprintf("The targetOwner:%s doesn't have data:%s yet, please double check.", targetOwner, targetDataName))
		}

		//for step 1, need to check whether the matching for these pair of data ever happened before, if Yes, just return with notice.
		//define queryResult(but not queryResults), due to queryByStepAndOperationType will return only one single result.
		queryResult, err := queryByStepAndOperationType(stub, operationType, false, 0, ownerId, dataName, targetOwner, targetDataName)
		if err != nil {
			return shim.Error(err.Error())
		}

		if queryResult != nil && len(queryResult) > 0 {
			var dataJSON OnBoarding
			err = json.Unmarshal(queryResult, &dataJSON)
			if err != nil {
				return shim.Error(err.Error())
			}

			//if the matching ever happened, but it is not finished(due to some reason), we should allow it to match again.
			if dataJSON.IsFinished {
				return shim.Error(fmt.Sprintf("This OnBoarding action already finished before, txID:%s", dataJSON.TxID))
			}
		}
	} else {
		//here means step > 1
		//check step, whether there is a (step - 1) happened before to make sure this is correct step. Also the step should not finished(isFinished==false)
		queryResult, err := queryByStepAndOperationType(stub, operationType, true, step - 1, ownerId, dataName, targetOwner, targetDataName)
		if err != nil {
			return shim.Error(err.Error())
		}
		if queryResult == nil || len(queryResult) == 0 {
			return shim.Error(fmt.Sprintf("Can not find the previous step:%d, can not continue.", step - 1))
		}

		var dataJSON OnBoarding
		err = json.Unmarshal(queryResult, &dataJSON)
		if err != nil {
			return shim.Error(err.Error())
		}

		txID = dataJSON.TxID	//reuse the txID of previous step
		if dataJSON.IsFinished {
			return shim.Error(fmt.Sprintf("This OnBoarding action already finished on step:%d, txID:%s", step - 1, txID))
		}
	}

	// === prepare the OnBoarding json ===
	txTimestamp, err := getTxTimestamp(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	data := &OnBoarding{operationType,
						txID,
						step,
						ownerId,
						dataName,
						filteredLineCount,
						targetOwner,
						targetDataName,
						isFinished,
						txTimestamp}

	dataJSONasBytes, err := json.Marshal(data)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === Save matching step to state ===
	key := operationType + "_" + txID
	fmt.Printf("Starting PutState, key:%s, value:%s\n", key, string(dataJSONasBytes))
	err = stub.PutState(key, dataJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === Save data MatchCount and LastMatchTimestamp to state ===
	if isFinished == true {
		operationType = "DataRegister"
		queryResults, err := queryByDataAndOperationType(stub, operationType, targetOwner, targetDataName)
		if err != nil {
			return shim.Error(err.Error())
		}
		var queryResult_DataRegistering_Array QueryResult_DataRegistering_Array
		err = json.Unmarshal(queryResults, &queryResult_DataRegistering_Array)
		if err != nil {
			return shim.Error(err.Error())
		}
		if len(queryResult_DataRegistering_Array) == 0 {
			return shim.Error(fmt.Sprintf("The targetDataName:%s belongs to targetOwner:%s doesn't exist.", targetDataName, targetOwner))
		}
		if len(queryResult_DataRegistering_Array) > 1 {
			return shim.Error(fmt.Sprintf("The targetDataName:%s belongs to targetOwner:%s has duplicated records.", targetDataName, targetOwner))
		}

		key = queryResult_DataRegistering_Array[0].Key

		record := queryResult_DataRegistering_Array[0].Record
		record.MatchCount = record.MatchCount + 1
		record.LastMatchTimestamp = txTimestamp
		dataJSONasBytes, err = json.Marshal(record)
		if err != nil {
			return shim.Error(err.Error())
		}
		fmt.Printf("Starting PutState, key:%s, value:%s\n", key, string(dataJSONasBytes))
		err = stub.PutState(key, dataJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	}

	return shim.Success(nil)
}


// ============================================================================================================================
// Query - query who am I, if I already registered, return the registered record, otherwise return nil.
// ============================================================================================================================
func (t *AdChainChaincode) WhoAmI(stub shim.ChaincodeStubInterface) pb.Response {

	ownerId, err := generateOwnerIdByCert(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	operationType := "OrgRegister"
	emptyQueryResults := []byte("[]")

	//If the ownerId already registered before, just return the history registered info.
	queryResults, err := queryByOwnerAndOperationType(stub, operationType, ownerId)
	if err != nil {
		return shim.Error(err.Error())
	}
	if bytes.Equal(queryResults[:], emptyQueryResults[:]) == false {
		return shim.Success(queryResults)
	}
	return shim.Success(nil)
}

// ============================================================================================================================
// Query - query a generic variable from ledger with complex query string in JSON format.
// ============================================================================================================================
func (t *AdChainChaincode) Query(stub shim.ChaincodeStubInterface) pb.Response {
	_, args := stub.GetFunctionAndParameters()
	var err error
	fmt.Println("starting Query")

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
		return nil, errors.New("Incorrect ownerId. Expecting 16 bytes of md5 hash which has len == 32 of hex string.")
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
		return nil, errors.New("Incorrect ownerId. Expecting 16 bytes of md5 hash which has len == 32 of hex string.")
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
	byStep bool, // if this is 'true', mean query by step, otherwise will not query by step.
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
	if byStep && step < 1 {
		return nil, errors.New("Incorrect step. Expecting step >= 1.")
	}
	if len(ownerId) != 32 || len(targetOwner) != 32 {
		return nil, errors.New(fmt.Sprintf("Incorrect owner or targetOwner. Expecting 16 bytes of md5 hash which has len == 32 of hex string.ownerId:%s", ownerId))
	}
	if len(dataName) == 0 || len(targetDataName) == 0 {
		return nil, errors.New("Incorrect dataName or targetDataName. Expecting non empty dataName and targetDataName.")
	}

	var queryString string
	if byStep == true {
		queryString = fmt.Sprintf("{\"selector\":{\"operationType\":\"%s\",\"step\":%d,\"owner\":\"%s\",\"dataName\":\"%s\",\"targetOwner\":\"%s\",\"targetDataName\":\"%s\"}}",
			operationType, step, ownerId, dataName, targetOwner, targetDataName)
	} else {
		queryString = fmt.Sprintf("{\"selector\":{\"operationType\":\"%s\",\"owner\":\"%s\",\"dataName\":\"%s\",\"targetOwner\":\"%s\",\"targetDataName\":\"%s\"}}",
			operationType, ownerId, dataName, targetOwner, targetDataName)
	}

	fmt.Printf("- queryByStepAndOperationType queryString:\n%s\n", queryString)

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

	fmt.Printf("- queryByStepAndOperationType queryResult:\n%s\n", queryResponse.Value)
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

// ============================================================================================================================
// Generate the ownerId which is the md5 hash value of cert
// ============================================================================================================================
func generateOwnerIdByCert(stub shim.ChaincodeStubInterface) (string, error) {
	idBytes, err := getCert(stub)
	if err != nil {
		return "", err
	}

	ownerId, err := md5_hash(idBytes)
	if err != nil {
		return "", err
	}
	return ownerId, nil
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
func getOrgNameAndCommonName(stub shim.ChaincodeStubInterface) (string, string, error) {

	idBytes, err := getCert(stub)
	if err != nil {
		return "", "", err
	}

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

	commonName := cert.Subject.CommonName
	if orgName == "" && commonName == "" {
		return "", "", errors.New("Both orgName amd commonName are empty.")
	}
	return orgName, commonName, nil
}

// ===============================================New support for panel========================================================
//
// ============================================================================================================================
// PanelRequest will happen when C has a requirement to calculate TA based on Provider A and Provider B. (C might be A or B)
// If the PanelRequest is already finished before, nothing will happen here.
// ============================================================================================================================
func (t *AdChainChaincode) PanelRequest(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	//-------------4 parameters------------
	//     0       		1       	 	  2		 			      3
	// "DataType"  "DataName"   "ProviderId_1|ProviderId_2"     "Tag"

	// ==== Input sanitation ====
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 5 parameters for PanelRequest")
	}
	//if there is any empty string parameters, return err.
	for i := 0; i < len(args); i++ {
		if len(args[i]) <= 0 {
			return shim.Error(strconv.Itoa(i) + "th argument must be a non-empty string")
		}
	}

	var txID string = stub.GetTxID() // the txID is important, it will be used to track all the steps of this paneling.

	ownerId, err := generateOwnerIdByCert(stub) // the ownerId here is the Sponsor of panel
	if err != nil {
		return shim.Error(err.Error())
	}

	operationType := function
	emptyQueryResults := []byte("[]")

	dataType := strings.ToLower(args[0])
	dataName := args[1]

	providerIdList := strings.Split(args[2], "|")
	tag := strings.ToLower(args[3])

    if len(providerIdList) != 2 {
		return shim.Error("Incorrect Providers argument. Expecting 2 providerIds for PanelRequest")
	}

	////Providers should not be same one
	//if providerIdList[0] == providerIdList[1] {
	//	return shim.Error("Incorrect Providers argument. Expecting 2 different providerIds for PanelRequest, now they are the same.")
	//}

	//check whether the owner exists, whether providers exists
	queryResults, err := queryByOwnerAndOperationType(stub, "OrgRegister", ownerId)
	if err != nil {
		return shim.Error(err.Error())
	}
	if bytes.Equal(queryResults[:], emptyQueryResults[:]) {
		return shim.Error(fmt.Sprintf("Current owner:%s has not registered yet, please do OrgRegister first.", ownerId))
	}

	for i := 0; i < len(providerIdList); i++ {
		queryResults, err := queryByOwnerAndOperationType(stub, "OrgRegister", providerIdList[i])
		if err != nil {
			return shim.Error(err.Error())
		}
		if bytes.Equal(queryResults[:], emptyQueryResults[:]) {
			return shim.Error(fmt.Sprintf("ProviderId:%s has not registered yet, please do OrgRegister first.", providerIdList[i]))
		}
	}

	//check whether the DataName(which belongs to ownerId) exists. Do not check the panel data exists or not here, because they will be checked inside onboarding.
	queryResults, err = queryByDataAndOperationType(stub, "DataRegister", ownerId, dataName)
	if err != nil {
		return shim.Error(err.Error())
	}
	if bytes.Equal(queryResults[:], emptyQueryResults[:]) {
		fmt.Sprintf("Current owner:%s doesn't have data:%s yet, please do DataRegister for this data first.\n", ownerId, dataName)
		return shim.Error(fmt.Sprintf("Current owner:%s doesn't have data:%s yet, please do DataRegister for this data first.", ownerId, dataName))
	}

	// === prepare the Paneling json ===
	var providers Providers
	//new object based on the tag, we might support other tags later here.
	if tag == "gender" {
		var genderProviderArray []GenderProvider
		genderProviderArray = append(genderProviderArray, GenderProvider{providerIdList[0], Gender{}})
		genderProviderArray = append(genderProviderArray, GenderProvider{providerIdList[1], Gender{}})
		providers = Providers{genderProviderArray}
	} else {
		return shim.Error(fmt.Sprintf("Current tag:%s has not been supported yet.", tag))
	}

	txTimestamp, err := getTxTimestamp(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	data := &Paneling{operationType,
		              		 txID,
							 ownerId,
							 dataType,
							 dataName,
		                     providers,
							 false,
							 txTimestamp,
							 pb_timestamp.Timestamp{0,0}} // lastMatchTimestamp is 0 when registering.

	dataJSONasBytes, err := json.Marshal(data)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === Save data to state ===
	key := operationType + "_" + txID
	fmt.Printf("Starting PutState, key:%s, value:%s\n", key, string(dataJSONasBytes))
	err = stub.PutState(key, dataJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

// ============================================================================================================================
// Query - query by operationType, txID for Paneling data.
// ============================================================================================================================
func queryByTxIDAndOperationType(stub shim.ChaincodeStubInterface, operationType string, txID string) ([]byte, error) {
	var err error
	fmt.Println("starting queryByTxIDAndOperationType")

	if len(operationType) == 0 {
		return nil, errors.New("Incorrect operationType. Expecting non empty type.")
	}
	if len(txID) == 0 {
		return nil, errors.New("Incorrect txID. Expecting non empty txID.")
	}

	queryString := fmt.Sprintf("{\"selector\":{\"operationType\":\"%s\",\"txID\":\"%s\"}}",
		operationType, txID)

	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()
	if resultsIterator.HasNext() == false {
		return nil, nil 	//there is no record for txID
	}
	queryResponse, err := resultsIterator.Next()
	if err != nil {
		return nil, err
	}

	fmt.Printf("- queryByTxIDAndOperationType queryResult:\n%s\n", queryResponse.Value)
	return queryResponse.Value, nil
}

// ============================================================================================================================
// PanelUpdate is used to update the PanelRequest submitted and not finished.
// If the PanelRequest is already finished before, nothing will happen here.
// ============================================================================================================================
func (t *AdChainChaincode) PanelUpdate(stub shim.ChaincodeStubInterface) pb.Response {
	_, args := stub.GetFunctionAndParameters()
	//-------------6 parameters------------
	//     0       		1       	        2		 	                   3			                 4
	//   "TxID"	    "IsFinished"  "Tag|Field|LineCount|HLL"     "Tag|Field|LineCount|HLL"       ...(add as many as we have)
	//Because the Paneling request might be triggered by the same Sponsor with same Data together with Same Providers multiple times. So TxID is the unique ID.

	// ==== Input sanitation ====
	if len(args) < 3 {
		return shim.Error("Incorrect number of arguments. Expecting at least 3 parameters for PanelUpdate")
	}
	//if there is any empty string parameters, return err.
	for i := 0; i < len(args); i++ {
		if len(args[i]) <= 0 {
			return shim.Error(strconv.Itoa(i) + "th argument must be a non-empty string")
		}
	}

	providerId, err := generateOwnerIdByCert(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	txID := args[0]
	isFinished, err := strconv.ParseBool(args[1])
	if err != nil {
		return shim.Error("2nd argument must be a boolean as isFinished of PanelUpdate.")
	}

	//check whether the Paneling record which has TxID exists.
	var dataJSON Paneling
	queryResult, err := queryByTxIDAndOperationType(stub, "PanelRequest", txID)
	if err != nil {
		return shim.Error(err.Error())
	}

	if queryResult != nil && len(queryResult) > 0 {
		err = json.Unmarshal(queryResult, &dataJSON)
		if err != nil {
			return shim.Error(err.Error())
		}
		//if the paneling ever happened and isFinished, should return.
		if dataJSON.IsFinished {
			return shim.Error(fmt.Sprintf("This Paneling action already finished before, txID:%s", dataJSON.TxID))
		}
	} else {
		fmt.Sprintf("Paneling data with TxID:%s doesn't exist, please do PanelRequest first.\n", txID)
		return shim.Error(fmt.Sprintf("Paneling data with TxID:%s doesn't exist, please do PanelRequest first.", txID))
	}

	var genderProvider_P *GenderProvider
	for i := 0; i < len(dataJSON.Providers.GenderProviderArray) ; i++ {
		if providerId == dataJSON.Providers.GenderProviderArray[i].ProviderId {
			genderProvider_P = &dataJSON.Providers.GenderProviderArray[i]
		}
	}

	if genderProvider_P == nil {
		fmt.Sprintf("Current owner:%s is not a provider in Paneling data which has txID:%s.\n", providerId, txID)
		return shim.Error(fmt.Sprintf("Current owner:%s is not a provider in Paneling data which has txID:%s.", providerId, txID))
	}

	for i := 2; i < len(args); i++ {
		list := strings.Split(args[i], "|")
		if len(list) != 4 {
			return shim.Error(strconv.Itoa(i) + "th argument must be combined by: Tag|Field|LineCount|HLL")
		}
		tag := strings.ToLower(list[0])
		field := strings.ToLower(list[1])
		lineCount, err := strconv.Atoi(list[2])
		if err != nil {
			return shim.Error(strconv.Itoa(i) + "th argument must contain a numeric string as lineCount.")
		}
		hll := strings.ToLower(list[3])
		if tag == "gender" {
			switch field {
			case "male":
				genderProvider_P.Gender.Male = DataDigest{lineCount, hll}
			case "female":
				genderProvider_P.Gender.Female = DataDigest{lineCount, hll}
			case "all":
				genderProvider_P.Gender.All = DataDigest{lineCount, hll}
			default:
				//should not go here
				return shim.Error(strconv.Itoa(i) + "th argument must contain an existing field name.")
			}
		} else {
			return shim.Error(strconv.Itoa(i) + "th argument must contain a valid tag.")
		}

	}

	// === prepare the Paneling json ===
	lastUpdatedTimestamp, err := getTxTimestamp(stub)
	if err != nil {
		return shim.Error(err.Error())
	}

	dataJSON.LastUpdatedTimestamp = lastUpdatedTimestamp
	dataJSON.IsFinished = isFinished

	dataJSONasBytes, err := json.Marshal(dataJSON)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === Save data to state ===
	key := "PanelRequest" + "_" + txID
	fmt.Printf("Starting PutState, key:%s, value:%s\n", key, string(dataJSONasBytes))
	err = stub.PutState(key, dataJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}


func main() {
	err := shim.Start(new(AdChainChaincode))
	if err != nil {
		fmt.Printf("Error starting AdChainChaincode: %s", err)
	}
}