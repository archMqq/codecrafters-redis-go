package resppars

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"
)

func ParseData(data []byte) ([]string, error) {
	str := strings.Split(string(data), "\r\n")
	if len(str) < 3 {
		return nil, errors.New("incorrect data length")
	}

	if !strings.HasPrefix(str[0], "*") {
		return nil, errors.New("missing array prefix")
	}
	num, err := strconv.Atoi(str[0][1:])
	if err != nil {
		return nil, errors.New("invalid array length")
	}

	res := make([]string, 0, num)
	idx := 1
	for i := 0; i < num && idx+1 < len(str); i++ {
		if !strings.HasPrefix(str[idx], "$") {
			return nil, errors.New("expected bulk string prefix")
		}
		bulkLen, err := strconv.Atoi(str[idx][1:])
		if err != nil || bulkLen < 0 {
			return nil, errors.New("invalid bulk string length")
		}
		idx++
		if idx >= len(str) {
			return nil, errors.New("unexpected end of data")
		}
		res = append(res, str[idx])
		idx++
	}

	return res, nil
}

func ParseCommand(reader *bufio.Reader) ([]string, error) {
	var num int
	var res []string
	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			return res, err
		}

		str = strings.TrimSpace(str)
		if len(str) == 0 {
			return res, nil
		}
		switch str[0] {
		case '*':
			num, err = strconv.Atoi(string((str)[1:]))
			if err != nil {
				return nil, err
			}
			res = make([]string, 0, num)
		case '$':
			bulkLen, err := strconv.Atoi(str[1:])
			if err != nil {
				return nil, err
			}
			if bulkLen < 0 {
				return res, err
			}

			data := make([]byte, bulkLen+2)
			_, err = io.ReadFull(reader, data)
			if err != nil {
				return nil, err
			}
			res = append(res, string(data[:bulkLen]))
			if len(res) == num {
				return res, nil
			}
		}
	}
}
