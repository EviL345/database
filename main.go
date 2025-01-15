package main

import (
	"context"
	"fmt"
	"github.com/EviL345/database/config"
	"github.com/EviL345/database/pkg/api"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
)

type server struct {
	api.UnimplementedApiServer
	db *pgxpool.Pool
}

func newServer(db *pgxpool.Pool) *server {
	return &server{
		db: db,
	}
}

func (s *server) GetList(ctx context.Context, _ *api.GetListRequest) (*api.GetListResponse, error) {
	query := "SELECT id, title, text, created_at, updated_at, done FROM task"

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("database.server.GetList: %w", err)
	}

	tasks := []*api.Task{}
	for rows.Next() {
		task := &api.Task{}
		if err = rows.Scan(&task.Id, &task.Title, &task.Text, &task.CreatedAt, &task.UpdatedAt, &task.Done); err != nil {
			return nil, fmt.Errorf("database.server.GetList: %w", err)
		}
		tasks = append(tasks, task)
	}

	return &api.GetListResponse{
		Tasks: tasks,
	}, nil
}

func (s *server) CreateTask(ctx context.Context, req *api.CreateTaskRequest) (*api.CreateTaskResponse, error) {
	query := "INSERT INTO task (title, text, created_at, updated_at) VALUES ($1, $2, $3, $4) RETURNING id"
	//
	row := s.db.QueryRow(ctx, query, req.Title, req.Text, req.CreatedAt, req.UpdatedAt)
	var id int64

	row.Scan(&id)

	return &api.CreateTaskResponse{Id: id}, nil
}

func (s *server) DoneTask(ctx context.Context, req *api.DoneTaskRequest) (*emptypb.Empty, error) {
	query := "UPDATE task SET done = $1 WHERE id = $2"

	if _, err := s.db.Exec(ctx, query, true, req.Id); err != nil {
		return nil, fmt.Errorf("database.server.DoneTask: %w", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *server) DeleteTask(ctx context.Context, req *api.DeleteTaskRequest) (*emptypb.Empty, error) {
	query := "DELETE FROM task WHERE id = $1"

	if _, err := s.db.Exec(ctx, query, req.Id); err != nil {
		return nil, fmt.Errorf("database.server.DeleteTask: %w", err)
	}

	return &emptypb.Empty{}, nil
}

func main() {
	cfg := config.GetConfig()
	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", cfg.Db.Host, cfg.Db.Port, cfg.Db.User, cfg.Db.Password, cfg.Db.Name)
	dbpool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	if err = dbpool.Ping(context.Background()); err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}

	log.Println("Connected to database")

	db, err := goose.OpenDBWithDriver("pgx", dbpool.Config().ConnString())
	if err != nil {
		log.Fatalf("Unable to open connection for up and down migrations: %v\n", err)
	}

	if err = goose.Up(db, "./migrations"); err != nil {
		log.Fatalf("Unable to run migrations: %v\n", err)
	}

	srv := newServer(dbpool)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Srv.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	reflection.Register(s)

	api.RegisterApiServer(s, srv)

	log.Printf("server listening at %v", lis.Addr())

	if err = s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
