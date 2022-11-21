package ruleengine_test

import (
	"go-iot/ruleengine"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTirgger(t *testing.T) {
	trigger := ruleengine.Trigger{
		FilterType: "event",
		Filters: []ruleengine.ConditionFilter{
			{Logic: "and", Key: "age", Operator: "eq", Value: "1"},
			{Logic: "or", Key: "age", Operator: "eq", Value: "2"},
			{Logic: "and", Key: "name", Operator: "eq", Value: "3"},
		},
	}
	assert.Equal(t, "age == 1 || (age == 2 && name == 3)", trigger.GetExpression())

	trigger = ruleengine.Trigger{
		FilterType: "event",
		Filters: []ruleengine.ConditionFilter{
			{Logic: "and", Key: "age", Operator: "eq", Value: "1"},
			{Logic: "or", Key: "age", Operator: "eq", Value: "2"},
			{Logic: "and", Key: "name", Operator: "eq", Value: "3"},
			{Logic: "or", Key: "age", Operator: "eq", Value: "2"},
		},
	}
	assert.Equal(t, "age == 1 || (age == 2 && name == 3) || (age == 2)", trigger.GetExpression())

	trigger = ruleengine.Trigger{
		FilterType: "event",
		Filters: []ruleengine.ConditionFilter{
			{Logic: "and", Key: "age", Operator: "eq", Value: "1"},
			{Logic: "and", Key: "age", Operator: "eq", Value: "2"},
			{Logic: "and", Key: "name", Operator: "eq", Value: "3"},
			{Logic: "or", Key: "age", Operator: "eq", Value: "2"},
		},
	}
	assert.Equal(t, "age == 1 && age == 2 && name == 3 || (age == 2)", trigger.GetExpression())

	trigger = ruleengine.Trigger{
		FilterType: "event",
		Filters: []ruleengine.ConditionFilter{
			{Logic: "and", Key: "age", Operator: "eq", Value: "1"},
			{Logic: "and", Key: "age", Operator: "eq", Value: "2"},
			{Logic: "and", Key: "name", Operator: "eq", Value: "3"},
			{Logic: "or", Key: "age", Operator: "eq", Value: "2"},
			{Logic: "or", Key: "age", Operator: "eq", Value: "3"},
		},
	}
	assert.Equal(t, "age == 1 && age == 2 && name == 3 || (age == 2) || (age == 3)", trigger.GetExpression())
}
