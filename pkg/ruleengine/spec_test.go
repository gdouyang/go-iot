package ruleengine_test

import (
	"go-iot/pkg/ruleengine"
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
	assert.Equal(t, "this.age == 1 || (this.age == 2 && this.name == 3)", trigger.GetExpression())
	data := map[string]interface{}{}
	data["age"] = 2
	data["name"] = 3
	res, err := trigger.Evaluate(data)
	assert.Nil(t, err)
	assert.True(t, res)

	trigger = ruleengine.Trigger{
		FilterType: "event",
		Filters: []ruleengine.ConditionFilter{
			{Logic: "and", Key: "age", Operator: "eq", Value: "1"},
			{Logic: "or", Key: "age", Operator: "eq", Value: "2"},
			{Logic: "and", Key: "name", Operator: "eq", Value: "3"},
			{Logic: "or", Key: "age", Operator: "eq", Value: "2"},
		},
	}
	assert.Equal(t, "this.age == 1 || (this.age == 2 && this.name == 3) || (this.age == 2)", trigger.GetExpression())
	data = map[string]interface{}{}
	data["age"] = 2
	data["name"] = 1
	res, err = trigger.Evaluate(data)
	assert.Nil(t, err)
	assert.True(t, res)

	trigger = ruleengine.Trigger{
		FilterType: "event",
		Filters: []ruleengine.ConditionFilter{
			{Logic: "and", Key: "age", Operator: "eq", Value: "1"},
			{Logic: "and", Key: "age", Operator: "eq", Value: "2"},
			{Logic: "and", Key: "name", Operator: "eq", Value: "3"},
			{Logic: "or", Key: "age", Operator: "eq", Value: "2"},
		},
	}
	assert.Equal(t, "this.age == 1 && this.age == 2 && this.name == 3 || (this.age == 2)", trigger.GetExpression())
	data = map[string]interface{}{}
	data["age"] = 3
	data["name"] = 1
	res, err = trigger.Evaluate(data)
	assert.Nil(t, err)
	assert.False(t, res)

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
	assert.Equal(t, "this.age == 1 && this.age == 2 && this.name == 3 || (this.age == 2) || (this.age == 3)", trigger.GetExpression())
	data = map[string]interface{}{}
	data["age"] = 3
	data["name"] = 1
	res, err = trigger.Evaluate(data)
	assert.Nil(t, err)
	assert.True(t, res)

	trigger = ruleengine.Trigger{
		FilterType: "event",
		Filters: []ruleengine.ConditionFilter{
			{Logic: "and", Key: "a", Operator: "eq", Value: "1"},
			{Logic: "and", Key: "b", Operator: "eq", Value: "2"},
			{Logic: "or", Key: "a", Operator: "eq", Value: "2"},
			{Logic: "or", Key: "a", Operator: "eq", Value: "3"},
			{Logic: "and", Key: "b", Operator: "eq", Value: "3"},
		},
	}
	assert.Equal(t, "this.a == 1 && this.b == 2 || (this.a == 2) || (this.a == 3 && this.b == 3)", trigger.GetExpression())
	data = map[string]interface{}{}
	data["a"] = 1
	data["b"] = 2
	res, err = trigger.Evaluate(data)
	assert.Nil(t, err)
	assert.True(t, res)

	trigger = ruleengine.Trigger{
		FilterType: "event",
		Filters: []ruleengine.ConditionFilter{
			{Logic: "and", Key: "a", Operator: "eq", Value: "1"},
			{Logic: "and", Key: "b", Operator: "eq", Value: "2"},
		},
	}
	assert.Equal(t, "this.a == 1 && this.b == 2", trigger.GetExpression())
	data = map[string]interface{}{}
	data["a"] = 1
	data["b"] = 2
	res, err = trigger.Evaluate(data)
	assert.Nil(t, err)
	assert.True(t, res)

	trigger = ruleengine.Trigger{
		FilterType: "event",
		Filters: []ruleengine.ConditionFilter{
			{Logic: "and", Key: "a", Operator: "eq", Value: "1"},
			{Logic: "and", Key: "b", Operator: "eq", Value: "2"},
		},
	}
	assert.Equal(t, "this.a == 1 && this.b == 2", trigger.GetExpression())
	data = map[string]interface{}{}
	data["a"] = "1"
	data["b"] = 2
	res, err = trigger.Evaluate(data)
	assert.Nil(t, err)
	assert.True(t, res)

	trigger = ruleengine.Trigger{
		FilterType: "event",
		Filters: []ruleengine.ConditionFilter{
			{Logic: "and", Key: "a", Operator: "eq", Value: "'aa'"},
			{Logic: "and", Key: "b", Operator: "eq", Value: "2"},
		},
	}
	assert.Equal(t, "this.a == 'aa' && this.b == 2", trigger.GetExpression())
	data = map[string]interface{}{}
	data["a"] = "aa"
	data["b"] = 2
	res, err = trigger.Evaluate(data)
	assert.Nil(t, err)
	assert.True(t, res)

	trigger = ruleengine.Trigger{
		FilterType: "event",
		Filters: []ruleengine.ConditionFilter{
			{Logic: "and", Key: "a", Operator: "eq", Value: "1"},
			{Logic: "and", Key: "b", Operator: "eq", Value: "2"},
		},
	}
	assert.Equal(t, "this.a == 1 && this.b == 2", trigger.GetExpression())
	data = map[string]interface{}{}
	data["a"] = "1a"
	data["b"] = 2
	res, err = trigger.Evaluate(data)
	assert.Nil(t, err)
	assert.False(t, res)
}
