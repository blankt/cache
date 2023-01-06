package consistenthash

import (
	"strconv"
	"testing"
)

func TestNewMap(t *testing.T) {
	hash := New(3, func(key []byte) uint32 {
		i, _ := strconv.Atoi(string(key))
		return uint32(i)
	})

	// 6 16 26 / 4 14 24  /2 12 22 --> 排序后 2 4 6 12 14 16 22 24 26
	hash.Add("6", "4", "2")

	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("获取节点出错 k:%v v:%v", k, v)
		}
	}

	//8, 18, 28
	hash.Add("8")

	testCases["27"] = "8"

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("获取节点出错 k:%v v:%v", k, v)
		}
	}
}
