package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

	"google.golang.org/grpc/credentials"

	"google.golang.org/grpc"

	"github.com/sc7639/31-grpc/todo"
)

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "missing subcommand: list or add")
		os.Exit(1)
	}

	// Load environment details from dot env file (.env)
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatalf("could not load dot env file: %v", err)
	}

	cert := os.Getenv("CERT_FILE")
	srvName := os.Getenv("SERVER_NAME")

	creds, err := credentials.NewClientTLSFromFile(cert, srvName)
	if err != nil {
		log.Fatalf("could not get client credentials: %v", err)
	}

	conn, err := grpc.Dial(":8888", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("could not connet to backend: %v", err)
	}
	client := todo.NewTasksClient(conn)

	switch cmd := flag.Arg(0); cmd {
	case "list":
		err = list(context.Background(), client)
	case "add":
		err = add(context.Background(), client, strings.Join(flag.Args()[1:], " "))
	case "complete":
		i, err := strconv.ParseInt(flag.Arg(1), 10, 32)
		if err != nil {
			break
		}
		err = complete(context.Background(), client, int32(i))
	default:
		err = fmt.Errorf("unkown sub command %s", cmd)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func add(ctx context.Context, client todo.TasksClient, text string) error {
	_, err := client.Add(ctx, &todo.AddReq{Text: text})
	if err != nil {
		return fmt.Errorf("could not add task in the backend: %v", err)
	}

	fmt.Println("task added successfully")
	return nil
}

func list(ctx context.Context, client todo.TasksClient) error {
	l, err := client.List(ctx, &todo.Void{})
	if err != nil {
		return fmt.Errorf("could not fetch tasks: %v", err)
	}

	for _, t := range l.Tasks {
		if t.Done {
			fmt.Printf("👍")
		} else {
			fmt.Printf("😱")
		}
		fmt.Printf(" %d: %s\n", t.Id, t.Text)
	}

	return nil
}

func complete(ctx context.Context, client todo.TasksClient, id int32) error {
	_, err := client.Complete(ctx, &todo.Id{Id: id})
	if err != nil {
		return fmt.Errorf("could not complete task: %v", err)
	}

	fmt.Println("task completed successfully")
	return nil
}
