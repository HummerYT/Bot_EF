package telegram

import (
	"database/sql"
	_ "github.com/lib/pq"
	"log"
)

var db *sql.DB

func InitDB(dataSourceName string) {
	var err error
	db, err = sql.Open("postgres", dataSourceName)
	if err != nil {
		log.Panic(err)
	}

	if err = db.Ping(); err != nil {
		log.Panic(err)
	}

	log.Println("Successfully connected to the database")
}

func GetPhysicsTask(section, difficulty string) (string, int64, error) {
	var task string
	var answer int64
	err := db.QueryRow("SELECT task, answer FROM physics_tasks WHERE section = $1 AND difficulty = $2 ORDER BY RANDOM() LIMIT 1", section, difficulty).Scan(&task, &answer)
	if err != nil {
		return "", 0, err
	}
	return task, answer, nil
}
