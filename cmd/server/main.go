package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/grpc/credentials"

	"github.com/golang/protobuf/proto"
	"github.com/sc7639/31-grpc/todo"
	grpc "google.golang.org/grpc"
)

func main() {
	// Load environment details from dot env file (.env)
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("could not load dot env file: %v", err)
	}

	cert := os.Getenv("CERT_FILE")
	key := os.Getenv("KEY_FILE")

	creds, err := credentials.NewServerTLSFromFile(cert, key)
	if err != nil {
		log.Fatalf("could not get server credentials: %v", err)
	}

	srv := grpc.NewServer(grpc.Creds(creds))
	var tasks taskServer
	todo.RegisterTasksServer(srv, tasks)
	l, err := net.Listen("tcp", ":8888")
	if err != nil {
		log.Fatalf("could not listen to :8888: %v", err)
	}
	log.Fatal(srv.Serve(l))
}

type taskServer struct{}

type length int64

const (
	sizeOfLength = 8
	dbPath       = "mydb.pb"
)

var endianness = binary.LittleEndian

func (s taskServer) Add(ctx context.Context, req *todo.AddReq) (*todo.Task, error) {
	l := &todo.TaskList{
		Tasks: make([]*todo.Task, 0),
	}

	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		l, err = s.List(ctx, &todo.Void{})
		if err != nil {
			return nil, fmt.Errorf("could not get list of tasks: %v", err)
		}
	}

	task := &todo.Task{
		Id:   int32(len(l.Tasks) + 1),
		Text: req.Text,
		Done: req.Done,
	}

	b, err := proto.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("could not encode task %v", err)
	}

	f, err := os.OpenFile(dbPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %v", dbPath, err)
	}

	if err := binary.Write(f, endianness, length(len(b))); err != nil {
		return nil, fmt.Errorf("could not encode length of message: %v", err)
	}
	_, err = f.Write(b)
	if err != nil {
		return nil, fmt.Errorf("could not write task to file: %v", err)
	}

	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("could not close file %s: %v", dbPath, err)
	}

	return task, nil
}

func (s taskServer) List(ctx context.Context, void *todo.Void) (*todo.TaskList, error) {
	b, err := ioutil.ReadFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %v", dbPath, err)
	}

	var tasks todo.TaskList
	for {
		if len(b) == 0 {
			return &tasks, nil
		} else if len(b) < sizeOfLength {
			return nil, fmt.Errorf("remaning odd %d bytes, what to do?", len(b))
		}

		var l length
		if err := binary.Read(bytes.NewReader(b[:sizeOfLength]), endianness, &l); err != nil {
			return nil, fmt.Errorf("could not decode message length: %v", err)
		}

		b = b[sizeOfLength:]

		var task todo.Task
		if err := proto.Unmarshal(b[:l], &task); err != nil {
			return nil, fmt.Errorf("could not read task: %v", err)
		}

		b = b[l:]

		tasks.Tasks = append(tasks.Tasks, &task)
	}
}

func (s taskServer) Complete(ctx context.Context, id *todo.Id) (*todo.Task, error) {
	var task *todo.Task

	l, err := s.List(ctx, &todo.Void{})
	if err != nil {
		return nil, fmt.Errorf("could not get list of tasks: %v", err)
	}

	if err = os.Remove(dbPath); err != nil {
		return nil, fmt.Errorf("could not delete db: %v", err)
	}

	for _, t := range l.Tasks {
		if t.Id == id.Id {
			t.Done = true
			task = t
		}

		_, err = s.Add(ctx, &todo.AddReq{Text: t.Text, Done: t.Done})
		if err != nil {
			return nil, fmt.Errorf("could not add task: %v", err)
		}
	}

	return task, nil
}
