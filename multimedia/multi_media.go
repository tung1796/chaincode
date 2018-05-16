/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

// ====CHAINCODE EXECUTION SAMPLES (CLI) ==================

// ==== Invoke multimedias ====
// peer chaincode invoke -C myc1 -n multimedias -c '{"Args":["initmultimedia","multimedia1","xxx","35","tom"]}'
// peer chaincode invoke -C myc1 -n multimedias -c '{"Args":["initmultimedia","multimedia2","xxx","50","tom"]}'
// peer chaincode invoke -C myc1 -n multimedias -c '{"Args":["initmultimedia","multimedia3","xx","70","tom"]}'
// peer chaincode invoke -C myc1 -n multimedias -c '{"Args":["transfermultimedia","multimedia2","jerry"]}'
// peer chaincode invoke -C myc1 -n multimedias -c '{"Args":["transfermultimediasBasedOnlink","blue","jerry"]}'
// peer chaincode invoke -C myc1 -n multimedias -c '{"Args":["delete","multimedia1"]}'

// ==== Query multimedias ====
// peer chaincode query -C myc1 -n multimedias -c '{"Args":["readmultimedia","multimedia1"]}'
// peer chaincode query -C myc1 -n multimedias -c '{"Args":["getmultimediasByRange","multimedia1","multimedia3"]}'
// peer chaincode query -C myc1 -n multimedias -c '{"Args":["getHistoryFormultimedia","multimedia1"]}'

// Rich Query (Only supported if CouchDB is used as state database):
//   peer chaincode query -C myc1 -n multimedias -c '{"Args":["querymultimediasByOwner","tom"]}'
//   peer chaincode query -C myc1 -n multimedias -c '{"Args":["querymultimedias","{\"selector\":{\"owner\":\"tom\"}}"]}'

// INDEXES TO SUPPORT COUCHDB RICH QUERIES
//
// Indexes in CouchDB are required in order to make JSON queries efficient and are required for
// any JSON query with a sort. As of Hyperledger Fabric 1.1, indexes may be packaged alongside
// chaincode in a META-INF/statedb/couchdb/indexes directory. Each index must be defined in its own
// text file with extension *.json with the index definition formatted in JSON following the
// CouchDB index JSON syntax as documented at:
// http://docs.couchdb.org/en/2.1.1/api/database/find.html#db-index
//
// This multimedias02 example chaincode demonstrates a packaged
// index which you can find in META-INF/statedb/couchdb/indexes/indexOwner.json.
// For deployment of chaincode to production environments, it is recommended
// to define any indexes alongside chaincode so that the chaincode and supporting indexes
// are deployed automatically as a unit, once the chaincode has been installed on a peer and
// instantiated on a channel. See Hyperledger Fabric documentation for more details.
//
// If you have access to the your peer's CouchDB state database in a development environment,
// you may want to iteratively test various indexes in support of your chaincode queries.  You
// can use the CouchDB Fauxton interface or a command line curl utility to create and update
// indexes. Then once you finalize an index, include the index definition alongside your
// chaincode in the META-INF/statedb/couchdb/indexes directory, for packaging and deployment
// to managed environments.
//
// In the examples below you can find index definitions that support multimedias02
// chaincode queries, along with the syntax that you can use in development environments
// to create the indexes in the CouchDB Fauxton interface or a curl command line utility.
//

//Example hostname:port configurations to access CouchDB.
//
//To access CouchDB docker container from within another docker container or from vagrant environments:
// http://couchdb:5984/
//
//Inside couchdb docker container
// http://127.0.0.1:5984/

// Index for docType, owner.
// Note that docType and owner fields must be prefixed with the "data" wrapper
//
// Index definition for use with Fauxton interface
// {"index":{"fields":["data.docType","data.owner"]},"ddoc":"indexOwnerDoc", "name":"indexOwner","type":"json"}
//
// Example curl command line to define index in the CouchDB channel_chaincode database
// curl -i -X POST -H "Content-Type: application/json" -d "{\"index\":{\"fields\":[\"data.docType\",\"data.owner\"]},\"name\":\"indexOwner\",\"ddoc\":\"indexOwnerDoc\",\"type\":\"json\"}" http://hostname:port/myc1_multimedias/_index
//

// Index for docType, owner, size (descending order).
// Note that docType, owner and size fields must be prefixed with the "data" wrapper
//
// Index definition for use with Fauxton interface
// {"index":{"fields":[{"data.size":"desc"},{"data.docType":"desc"},{"data.owner":"desc"}]},"ddoc":"indexSizeSortDoc", "name":"indexSizeSortDesc","type":"json"}
//
// Example curl command line to define index in the CouchDB channel_chaincode database
// curl -i -X POST -H "Content-Type: application/json" -d "{\"index\":{\"fields\":[{\"data.size\":\"desc\"},{\"data.docType\":\"desc\"},{\"data.owner\":\"desc\"}]},\"ddoc\":\"indexSizeSortDoc\", \"name\":\"indexSizeSortDesc\",\"type\":\"json\"}" http://hostname:port/myc1_multimedias/_index

// Rich Query with index design doc and index name specified (Only supported if CouchDB is used as state database):
//   peer chaincode query -C myc1 -n multimedias -c '{"Args":["querymultimedias","{\"selector\":{\"docType\":\"multimedia\",\"owner\":\"tom\"}, \"use_index\":[\"_design/indexOwnerDoc\", \"indexOwner\"]}"]}'

// Rich Query with index design doc specified only (Only supported if CouchDB is used as state database):
//   peer chaincode query -C myc1 -n multimedias -c '{"Args":["querymultimedias","{\"selector\":{\"docType\":{\"$eq\":\"multimedia\"},\"owner\":{\"$eq\":\"tom\"},\"size\":{\"$gt\":0}},\"fields\":[\"docType\",\"owner\",\"size\"],\"sort\":[{\"size\":\"desc\"}],\"use_index\":\"_design/indexSizeSortDoc\"}"]}'

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

type multimedia struct {
	ObjectType string `json:"docType"` //docType is used to distinguish the various types of objects in state database
	Name       string `json:"name"`    //the fieldtags are needed to keep case from bouncing around
	link       string `json:"link"`
	Size       int    `json:"size"`
	cOwner     string `json:"oowner"`
	oOwner     string `json:"cOwner"` 
}

// ===================================================================================
// Main
// ===================================================================================
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init initializes chaincode
// ===========================
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke - Our entry point for Invocations
// ========================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "initMultimedia" { //create a new multimedia
		return t.initMultimedia(stub, args)
	} else if function == "transfermultimedia" { //change owner of a specific multimedia
		return t.transfermultimedia(stub, args)
	} else if function == "transfermultimediasBasedOnlink" { //transfer all multimedias of a certain link
		return t.transfermultimediasBasedOnlink(stub, args)
	} else if function == "delete" { //delete a multimedia
		return t.delete(stub, args)
	} else if function == "readmultimedia" { //read a multimedia
		return t.readmultimedia(stub, args)
	} else if function == "querymultimediasByOwner" { //find multimedias for owner X using rich query
		return t.querymultimediasByOwner(stub, args)
	} else if function == "querymultimedias" { //find multimedias based on an ad hoc rich query
		return t.querymultimedias(stub, args)
	} else if function == "getHistoryFormultimedia" { //get history of values for a multimedia
		return t.getHistoryFormultimedia(stub, args)
	} else if function == "getmultimediasByRange" { //get multimedias based on range query
		return t.getmultimediasByRange(stub, args)
	}

	fmt.Println("invoke did not find func: " + function) //error
	return shim.Error("Received unknown function invocation")
}

// ============================================================
// initmultimedia - create a new multimedia, store into chaincode state
// ============================================================
func (t *SimpleChaincode) initMultimedia(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	//   0       1       2     3	4
	// "asdf", "blue", "35", "bob","xx "
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
	}

	// ==== Input sanitation ====
	fmt.Println("- start init multimedia")
	if len(args[0]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("3rd argument must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return shim.Error("4th argument must be a non-empty string")
	}
	multimediaName := args[0]
	link := strings.ToLower(args[1])
	cOwner := strings.ToLower(args[3])
	oowner := strings.ToLower(args[4])
	size, err := strconv.Atoi(args[2])
	if err != nil {
		return shim.Error("3rd argument must be a numeric string")
	}

	// ==== Check if multimedia already exists ====
	multimediaAsBytes, err := stub.GetState(multimediaName)
	if err != nil {
		return shim.Error("Failed to get multimedia: " + err.Error())
	} else if multimediaAsBytes != nil {
		fmt.Println("This multimedia already exists: " + multimediaName)
		return shim.Error("This multimedia already exists: " + multimediaName)
	}

	// ==== Create multimedia object and marshal to JSON ====
	objectType := "multimedia" // khai bao 1 object ten multimedia
	multimedia := &multimedia{objectType, multimediaName, link, size, cowner,oOwner}
	multimediaJSONasBytes, err := json.Marshal(multimedia)// encode data json
	if err != nil {
		return shim.Error(err.Error())
	}
	//Alternatively, build the multimedia json string manually if you don't want to use struct marshalling
	//multimediaJSONasString := `{"docType":"multimedia",  "name": "` + multimediaName + `", "link": "` + link+ `", "size": ` + strconv.Itoa(size) + `, "owner": "` + owner + `"}`
	//multimediaJSONasBytes := []byte(str)

	// === Save multimedia to state ===
	err = stub.PutState(multimediaName, multimediaJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	//  ==== Index the multimedia to enable link-based range queries, e.g. return all blue multimedias ====
	//  An 'index' is a normal key/value entry in state.
	//  The key is a composite key, with the elements that you want to range query on listed first.
	//  In our case, the composite key is based on indexName~link~name.
	//  This will enable very efficient state range queries based on composite keys matching indexName~link~*
	indexName := "link~name"
	linkNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{multimedia.link, multimedia.Name})
	if err != nil {
		return shim.Error(err.Error())
	}
	//  Save index entry to state. Only the key name is needed, no need to store a duplicate copy of the multimedia.
	//  Note - passing a 'nil' value will effectively delete the key from state, therefore we pass null character as value
	value := []byte{0x00}
	stub.PutState(linkNameIndexKey, value)

	// ==== multimedia saved and indexed. Return success ====
	fmt.Println("- end init multimedia")
	return shim.Success(nil)
}

// ===============================================
// readmultimedia - read a multimedia from chaincode state
// ===============================================
func (t *SimpleChaincode) readmultimedia(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the multimedia to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetState(name) //get the multimedia from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"multimedia does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsbytes)
}

// ==================================================
// delete - remove a multimedia key/value pair from state
// ==================================================
func (t *SimpleChaincode) delete(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var jsonResp string
	var multimediaJSON multimedia
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	multimediaName := args[0]

	// to maintain the link~name index, we need to read the multimedia first and get its link
	valAsbytes, err := stub.GetState(multimediaName) //get the multimedia from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + multimediaName + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"multimedia does not exist: " + multimediaName + "\"}"
		return shim.Error(jsonResp)
	}

	err = json.Unmarshal([]byte(valAsbytes), &multimediaJSON)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to decode JSON of: " + multimediaName + "\"}"
		return shim.Error(jsonResp)
	}

	err = stub.DelState(multimediaName) //remove the multimedia from chaincode state
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}

	// maintain the index
	indexName := "link~name"
	linkNameIndexKey, err := stub.CreateCompositeKey(indexName, []string{multimediaJSON.link, multimediaJSON.Name})
	if err != nil {
		return shim.Error(err.Error())
	}

	//  Delete index entry to state.
	err = stub.DelState(linkNameIndexKey)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}
	return shim.Success(nil)
}

// ===========================================================
// transfer a multimedia by setting a new owner name on the multimedia
// ===========================================================
func (t *SimpleChaincode) transfermultimedia(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0       1
	// "name", "bob"
	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	multimediaName := args[0]
	newOwner := strings.ToLower(args[1])
	fmt.Println("- start transfermultimedia ", multimediaName, newOwner)

	multimediaAsBytes, err := stub.GetState(multimediaName)
	if err != nil {
		return shim.Error("Failed to get multimedia:" + err.Error())
	} else if multimediaAsBytes == nil {
		return shim.Error("multimedia does not exist")
	}

	multimediaToTransfer := multimedia{}
	err = json.Unmarshal(multimediaAsBytes, &multimediaToTransfer) //unmarshal it aka JSON.parse()
	if err != nil {
		return shim.Error(err.Error())
	}
	multimediaToTransfer.oOwner +=","+multimediaToTransfer.cOwner
	multimediaToTransfer.cOwner = newOwner //change the owner

	multimediaJSONasBytes, _ := json.Marshal(multimediaToTransfer)
	err = stub.PutState(multimediaName, multimediaJSONasBytes) //rewrite the multimedia
	if err != nil {	
		return shim.Error(err.Error())
	}

	fmt.Println("- end transfermultimedia (success)")
	return shim.Success(nil)
}

// ===========================================================================================
// getmultimediasByRange performs a range query based on the start and end keys provided.

// Read-only function results are not typically submitted to ordering. If the read-only
// results are submitted to ordering, or if the query is used in an update transaction
// and submitted to ordering, then the committing peers will re-execute to guarantee that
// result sets are stable between endorsement time and commit time. The transaction is
// invalidated by the committing peers if the result set has changed between endorsement
// time and commit time.
// Therefore, range queries are a safe option for performing update transactions based on query results.
// ===========================================================================================
func (t *SimpleChaincode) getmultimediasByRange(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	startKey := args[0]
	endKey := args[1]

	resultsIterator, err := stub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
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

	fmt.Printf("- getmultimediasByRange queryResult:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

// ==== Example: GetStateByPartialCompositeKey/RangeQuery =========================================
// transfermultimediasBasedOnlink will transfer multimedias of a given link to a certain new owner.
// Uses a GetStateByPartialCompositeKey (range query) against link~name 'index'.
// Committing peers will re-execute range queries to guarantee that result sets are stable
// between endorsement time and commit time. The transaction is invalidated by the
// committing peers if the result set has changed between endorsement time and commit time.
// Therefore, range queries are a safe option for performing update transactions based on query results.
// ===========================================================================================
func (t *SimpleChaincode) transfermultimediasBasedOnlink(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0       1
	// "link", "bob"
	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	link := args[0]
	newOwner := strings.ToLower(args[1])
	fmt.Println("- start transfermultimediasBasedOnlink ", link, newOwner)

	// Query the link~name index by link
	// This will execute a key range query on all keys starting with 'link'
	linkedmultimediaResultsIterator, err := stub.GetStateByPartialCompositeKey("link~name", []string{link})
	if err != nil {
		return shim.Error(err.Error())
	}
	defer linkedmultimediaResultsIterator.Close()

	// Iterate through result set and for each multimedia found, transfer to newOwner
	var i int
	for i = 0; linkedmultimediaResultsIterator.HasNext(); i++ {
		// Note that we don't get the value (2nd return variable), we'll just get the multimedia name from the composite key
		responseRange, err := linkedmultimediaResultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		// get the link and name from link~name composite key
		objectType, compositeKeyParts, err := stub.SplitCompositeKey(responseRange.Key)
		if err != nil {
			return shim.Error(err.Error())
		}
		returnedlink := compositeKeyParts[0]
		returnedmultimediaName := compositeKeyParts[1]
		fmt.Printf("- found a multimedia from index:%s link:%s name:%s\n", objectType, returnedlink, returnedmultimediaName)

		// Now call the transfer function for the found multimedia.
		// Re-use the same function that is used to transfer individual multimedias
		response := t.transfermultimedia(stub, []string{returnedmultimediaName, newOwner})
		// if the transfer failed break out of loop and return error
		if response.Status != shim.OK {
			return shim.Error("Transfer failed: " + response.Message)
		}
	}

	responsePayload := fmt.Sprintf("Transferred %d %s multimedias to %s", i, link, newOwner)
	fmt.Println("- end transfermultimediasBasedOnlink: " + responsePayload)
	return shim.Success([]byte(responsePayload))
}

// =======Rich queries =========================================================================
// Two examples of rich queries are provided below (parameterized query and ad hoc query).
// Rich queries pass a query string to the state database.
// Rich queries are only supported by state database implementations
//  that support rich query (e.g. CouchDB).
// The query string is in the syntax of the underlying state database.
// With rich queries there is no guarantee that the result set hasn't changed between
//  endorsement time and commit time, aka 'phantom reads'.
// Therefore, rich queries should not be used in update transactions, unless the
// application handles the possibility of result set changes between endorsement and commit time.
// Rich queries can be used for point-in-time queries against a peer.
// ============================================================================================

// ===== Example: Parameterized rich query =================================================
// querymultimediasByOwner queries for multimedias based on a passed in owner.
// This is an example of a parameterized query where the query logic is baked into the chaincode,
// and accepting a single query parameter (owner).
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *SimpleChaincode) querymultimediasByOwner(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "bob"
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	owner := strings.ToLower(args[0])

	queryString := fmt.Sprintf("{\"selector\":{\"docType\":\"multimedia\",\"cOwner\":\"%s\"}}", owner)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

// ===== Example: Ad hoc rich query ========================================================
// querymultimedias uses a query string to perform a query for multimedias.
// Query string matching state database syntax is passed in and executed as is.
// Supports ad hoc queries that can be defined at runtime by the client.
// If this is not desired, follow the querymultimediasForOwner example for parameterized queries.
// Only available on state databases that support rich query (e.g. CouchDB)
// =========================================================================================
func (t *SimpleChaincode) querymultimedias(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "queryString"
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
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

func (t *SimpleChaincode) getHistoryFormultimedia(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	multimediaName := args[0]

	fmt.Printf("- start getHistoryFormultimedia: %s\n", multimediaName)

	resultsIterator, err := stub.GetHistoryForKey(multimediaName)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing historic values for the multimedia
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"TxId\":")
		buffer.WriteString("\"")
		buffer.WriteString(response.TxId)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Value\":")
		// if it was a delete operation on given key, then we need to set the
		//corresponding value null. Else, we will write the response.Value
		//as-is (as the Value itself a JSON multimedia)
		if response.IsDelete {
			buffer.WriteString("null")
		} else {
			buffer.WriteString(string(response.Value))
		}

		buffer.WriteString(", \"Timestamp\":")
		buffer.WriteString("\"")
		buffer.WriteString(time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).String())
		buffer.WriteString("\"")

		buffer.WriteString(", \"IsDelete\":")
		buffer.WriteString("\"")
		buffer.WriteString(strconv.FormatBool(response.IsDelete))
		buffer.WriteString("\"")

		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getHistoryFormultimedia returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}
