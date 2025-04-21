package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	bolt "go.etcd.io/bbolt"
)

type task struct {
	name  []byte
	notes []byte
}

type todoList struct {
	db    *bolt.DB
	tasks []*task
}

func (tl *todoList) initializeList() error {
	tl.tasks = []*task{}
	return tl.db.View(func(tx *bolt.Tx) error {
		tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			tl.tasks = append(tl.tasks, &task{name: b.Get([]byte("name")), notes: b.Get([]byte("notes"))})
			return nil
		})
		return nil
	})
}

func (tl *todoList) listTasks() {
	for _, task := range tl.tasks {
		log.Printf("Name: %s, Notes: %s", string(task.name), string(task.notes))
	}
}

func (t *task) writeTask(db *bolt.DB) error {
	err := db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(t.name)
		if b == nil || err != nil {
			return errors.New("error creating task")
		}
		err = b.Put([]byte("name"), t.name)
		if err != nil {
			return errors.New("error adding metadata")
		}
		err = b.Put([]byte("notes"), t.notes)
		if err != nil {
			return errors.New("error adding metadata")
		}
		return nil
	})
	return err
}

func (t *task) readTask(db *bolt.DB) error {
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(t.name))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Printf("key=%s, value=%s\n", k, v)
		}

		return nil
	})
	return err
}

func (t *task) removeTask(db *bolt.DB) error {
	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(t.name))
		return err
	})
	return err
}

func createTask(name string, notes string) *task {
	t := &task{name: []byte(name), notes: []byte(notes)}
	return t
}

func (tl *todoList) httpGetTodoList(w http.ResponseWriter, r *http.Request) {
	tl.initializeList()
	tl.listTasks()
}

func (tl *todoList) httpAddTask(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Errorf("error parsing form %s", err)
	}
	taskName := r.PostForm.Get("name")
	notes := r.PostForm.Get("notes")

	fmt.Println(taskName, notes)

	t := &task{name: []byte(taskName), notes: []byte(notes)}
	err = t.writeTask(tl.db)
	if err != nil {
		fmt.Errorf("error writing task to DB")
	}
}

func main() {
	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := chi.NewRouter()
	tl := &todoList{db: db}

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", tl.httpGetTodoList)
	r.Post("/add", tl.httpAddTask)

	err = http.ListenAndServe(":3000", r)
	if err != nil {
		log.Fatalf("Server Cannot Be Started: %s", err)
	}
}
