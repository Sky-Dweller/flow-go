// (c) 2019 Dapper Labs - ALL RIGHTS RESERVED

package operation

import (
	"github.com/dgraph-io/badger/v2"

	"github.com/dapperlabs/flow-go/model/flow"
)

// NOTE: These insert light collections, which only contain references
// to the constituent transactions. They do not modify transactions contained
// by the collections.

func InsertCollection(collection *flow.LightCollection) func(*badger.Txn) error {
	return insert(makePrefix(codeCollection, collection.ID()), collection)
}

func CheckCollection(collID flow.Identifier, exists *bool) func(*badger.Txn) error {
	return check(makePrefix(codeCollection, collID), exists)
}

func RetrieveCollection(collID flow.Identifier, collection *flow.LightCollection) func(*badger.Txn) error {
	return retrieve(makePrefix(codeCollection, collID), collection)
}

func RemoveCollection(collID flow.Identifier) func(*badger.Txn) error {
	return remove(makePrefix(codeCollection, collID))
}

// IndexCollection indexes the transactions within the collection payload of a block.
func IndexCollectionPayload(height uint64, blockID, parentID flow.Identifier, collection *flow.LightCollection) func(*badger.Txn) error {
	return insert(makePrefix(codeIndexCollection, height, blockID, parentID), collection.Transactions)
}

// LookupCollection looks up the collection for a given payload.
func LookupCollectionPayload(height uint64, blockID, parentID flow.Identifier, collection *flow.LightCollection) func(*badger.Txn) error {
	return retrieve(makePrefix(codeIndexCollection, height, blockID, parentID), &collection.Transactions)
}

// VerifyCollectionPayload verifies that a collection does not contain any
// transactions that ever been included in a payload in the block's ancestry.
func VerifyCollectionPayload(height uint64, blockID flow.Identifier, txIDs []flow.Identifier) func(*badger.Txn) error {
	return iterate(makePrefix(codeIndexCollection, height), makePrefix(codeIndexCollection, uint64(0)), verifypayload(blockID, txIDs))
}
