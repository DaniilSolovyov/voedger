/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

package query2

import (
	"encoding/json"
	"log"
	"testing"
)

func Test(t *testing.T) {
	body := `{"Year":2024, "Month":{"$in":[1,2,3]}}`

	obj := make(map[string]interface{})

	err := json.Unmarshal([]byte(body), &obj)
	if err != nil {
		panic(err)
	}

	log.Println(obj)
}
