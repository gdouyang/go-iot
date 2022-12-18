package util

import "strconv"

func StringToInt64(val string) (int64, error) {
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return int64(i), nil
}
