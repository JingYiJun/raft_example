package main

import (
	"fmt"
	"github.com/desertbit/grumble"
	"log"
	"net/rpc"
	"raft_example/raft"
)

func main() {
	var app = grumble.New(&grumble.Config{
		Name:        "raft_client",
		Description: "raft client",
	})

	app.AddCommand(&grumble.Command{
		Name: "get",
		Help: "get raft server info",
		Flags: func(f *grumble.Flags) {
			f.String("a", "addr", "", "server address")
		},
		Run: func(c *grumble.Context) error {
			addr := c.Flags.String("addr")
			if addr == "" {
				return fmt.Errorf("addr is empty")
			}
			log.Printf("get raft server info from: %v", addr)
			client, err := rpc.DialHTTP("tcp", addr)
			if err != nil {
				log.Printf("rpc.DialHTTP error: %v", err)
				return err
			}

			args := struct{}{}
			reply := &raft.ServerInfo{}
			err = client.Call("Raft.GetInfo", args, reply)
			if err != nil {
				log.Printf("client.Call error: %v", err)
				return err
			}

			log.Printf("server info: id [%v], term %d, value %d", reply.Id, reply.Term, reply.Value)

			return nil
		},
	})

	app.AddCommand(&grumble.Command{
		Name: "set",
		Help: "set raft server value",
		Flags: func(f *grumble.Flags) {
			f.String("a", "addr", "", "server address")
			f.Int("v", "value", 0, "server value")
		},
		Run: func(c *grumble.Context) error {
			addr := c.Flags.String("addr")
			if addr == "" {
				return fmt.Errorf("addr is empty")
			}
			log.Printf("get raft server info from: %v", addr)
			client, err := rpc.DialHTTP("tcp", addr)
			if err != nil {
				return err
			}

			args := c.Flags.Int("value")
			reply := &raft.CommonReply{}
			err = client.Call("Raft.Start", raft.StartArgs{Value: args}, reply)
			if err != nil {
				return err
			}

			log.Printf("server reply: %v", reply.Success)

			return nil
		},
	})

	err := app.Run()
	if err != nil {
		panic(err)
	}
}
