package eventbus_test

import (
	"fmt"
	"testing"
)

func TestMap(t *testing.T) {
	{
		var m1 map[string]string = map[string]string{}
		m1["a"] = "aaa"
		fmt.Println("m1:", m1)
		fmt.Printf("m1 point: %p \n", m1)
		t1(m1)
		fmt.Println("m1:", m1)

		var m21 map[string]string = map[string]string{}
		m21 = m1
		m21["a"] = "ccc"
		fmt.Println("m21:", m21)

		fmt.Println("m1:", m1)

		var m3 map[string]string
		fmt.Println("m3:", m3)
	}

	{
		var s1 []string = []string{"1"}
		s2 := s1
		s2[0] = "2"
		fmt.Println(s1, s2)
		fmt.Printf("s1 point: %p \n", s1)
		ss1(s1)
		fmt.Println(s1)
	}

	{
		var s1 string = "1"
		fmt.Printf("s1 point: %v \n", &s1)
		t0(s1)
	}
}

func t0(s2 string) {
	fmt.Printf("s2 point: %v \n", &s2)
}

func t1(m2 map[string]string) {
	m2["a"] = "bbb"
	fmt.Println("m2:", m2)
	fmt.Printf("m2 point: %p \n", m2)
}

func ss1(s2 []string) {
	fmt.Printf("s2 point: %p \n", s2)
	s3 := &s2
	(*s3) = append(*s3, "4")
}
