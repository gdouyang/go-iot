package product

import (
	"errors"
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestSaveScriptCompileBeforeWrite(t *testing.T) {
	var updated bool
	origGet := getProductMust
	origUpdate := updateProduct
	origCompile := compileOnly
	origNew := newCodec
	t.Cleanup(func() {
		getProductMust = origGet
		updateProduct = origUpdate
		compileOnly = origCompile
		newCodec = origNew
	})

	getProductMust = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CodecId: core.Script_Codec, CreateId: 7}}, nil
	}
	compileOnly = func(script string) error {
		return errors.New("SyntaxError: unexpected eof")
	}
	updateProduct = func(ob *models.ProductModel) error {
		updated = true
		return nil
	}
	newCodec = func(codecId, productId, script string) (core.Codec, error) {
		t.Fatal("NewCodec must not run when compile fails")
		return nil, nil
	}

	err := SaveScript(7, "P1", "function broken(")
	require.Error(t, err)
	require.Contains(t, err.Error(), "SyntaxError")
	require.False(t, updated, "compile failure must not persist script")
}

func TestSaveScriptRejectsNonScriptCodec(t *testing.T) {
	var updated bool
	origGet := getProductMust
	origUpdate := updateProduct
	t.Cleanup(func() {
		getProductMust = origGet
		updateProduct = origUpdate
	})
	getProductMust = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CodecId: "modbus-script-core", CreateId: 7}}, nil
	}
	updateProduct = func(ob *models.ProductModel) error {
		updated = true
		return nil
	}
	err := SaveScript(7, "P1", `function OnMessage(c){}`)
	require.ErrorIs(t, err, ErrNotScriptCodec)
	require.False(t, updated)
}

func TestSaveScriptOwnerCheck(t *testing.T) {
	origGet := getProductMust
	t.Cleanup(func() { getProductMust = origGet })
	getProductMust = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CodecId: core.Script_Codec, CreateId: 1}}, nil
	}
	err := SaveScript(2, "P1", `function OnMessage(c){}`)
	require.ErrorIs(t, err, ErrNotOwner)
}

func TestSaveTSLRejectsEmptyEnum(t *testing.T) {
	var updated bool
	origGet := getProductMust
	origUpdate := updateProduct
	t.Cleanup(func() {
		getProductMust = origGet
		updateProduct = origUpdate
	})
	getProductMust = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 7}}, nil
	}
	updateProduct = func(ob *models.ProductModel) error {
		updated = true
		return nil
	}
	err := SaveTSL(7, "P1", `{"properties":[{"id":"mode","name":"mode","type":"enum","elements":[]}]}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "enum elements is empty")
	require.False(t, updated)
}

func TestSaveScriptSuccessCompilesThenWritesThenNewCodec(t *testing.T) {
	var compiled, updated, codeced bool
	origGet := getProductMust
	origUpdate := updateProduct
	origCompile := compileOnly
	origNew := newCodec
	t.Cleanup(func() {
		getProductMust = origGet
		updateProduct = origUpdate
		compileOnly = origCompile
		newCodec = origNew
	})
	getProductMust = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CodecId: core.Script_Codec, CreateId: 7}}, nil
	}
	compileOnly = func(script string) error {
		compiled = true
		require.False(t, updated)
		return nil
	}
	updateProduct = func(ob *models.ProductModel) error {
		require.True(t, compiled)
		updated = true
		return nil
	}
	newCodec = func(codecId, productId, script string) (core.Codec, error) {
		require.True(t, updated)
		codeced = true
		return nil, nil
	}
	require.NoError(t, SaveScript(7, "P1", `function OnMessage(c){}`))
	require.True(t, codeced)
}
