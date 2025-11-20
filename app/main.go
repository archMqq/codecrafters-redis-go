package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/internal/cache"
	"github.com/codecrafters-io/redis-starter-go/app/internal/resppars"
	"github.com/xiam/resp"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment the code below to pass the first stage
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	cache := cache.NewCache(time.Second*5, time.Second*5)
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go func(con net.Conn) {
			reader := bufio.NewReader(conn)
			for {
				cmd, err := resppars.ParseCommand(reader)
				if err != nil && err != io.EOF {
					conn.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(err.Error()), err.Error())))
					continue
				}
				if err == io.EOF {
					return
				}
				switch strings.ToUpper(cmd[0]) {
				case "PING":
					conn.Write([]byte("+PONG\r\n"))
				case "ECHO":
					if len(cmd) < 2 {
						conn.Write([]byte("$0\r\n\r\n"))
						continue
					}
					res, err := resp.Marshal(cmd[1])
					if err != nil {
						conn.Write([]byte("-ERR internal error\r\n"))
						continue
					}
					conn.Write([]byte(res))
				case "SET":
					if len(cmd) < 3 {
						conn.Write([]byte("$0\r\n\r\n"))
						continue
					}
					if len(cmd) == 3 {
						err = cache.SetWithoutExp(cmd[1], cmd[2])
						if err != nil {
							conn.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(err.Error()), err.Error())))
							continue
						}
					} else {
						val, err := strconv.Atoi(cmd[4])
						if err != nil {
							conn.Write([]byte("-ERR time convert\r\n"))
							continue
						}

						var exp time.Duration
						switch strings.ToUpper(cmd[3]) {
						case "PX":
							exp = time.Millisecond * time.Duration(val)
						case "EX":
							exp = time.Second * time.Duration(val)
						}

						err = cache.SetWithExp(cmd[1], cmd[2], exp)
						if err != nil {
							conn.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(err.Error()), err.Error())))
							continue
						}
					}

					conn.Write([]byte("+OK\r\n"))
				case "GET":
					if len(cmd) < 2 {
						conn.Write([]byte("$0\r\n\r\n"))
						continue
					}
					val, err := cache.Get(cmd[1])
					if err != nil {
						conn.Write([]byte("$-1\r\n"))
						continue
					}
					res, err := resp.Marshal(val)
					if err != nil {
						conn.Write([]byte("-ERR internal error\r\n"))
						continue
					}
					conn.Write([]byte(res))
				case "RPUSH":
					if len(cmd) < 3 {
						conn.Write([]byte("$0\r\n\r\n"))
						continue
					}

					count := cache.RSet(cmd[1], cmd[2])
					conn.Write([]byte(fmt.Sprintf(":%d\r\n", count)))
				default:
					conn.Write([]byte("-ERR unknown command\r\n"))
				}
			}
		}(conn)

	}
}
