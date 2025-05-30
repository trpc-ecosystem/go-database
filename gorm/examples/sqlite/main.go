package main

import (
	tgorm "github.com/trpc-group/trpc-database/gorm"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
)

// User is the model struct.
type User struct {
	ID       int
	Username string
}

func main() {
	_ = trpc.NewServer()

	connPool := tgorm.NewConnPool("trpc.sqlite.server.service")
	cli, err := gorm.Open(&sqlite.Dialector{Conn: connPool}, &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Create record
	insertUser := User{ID: 10, Username: "gorm-client"}
	result := cli.Table("Users").Create(&insertUser)
	log.Infof("inserted data's primary key: %d, err: %v", insertUser.ID, result.Error)

	// Query record
	var queryUser User
	if err := cli.First(&queryUser).Error; err != nil {
		panic(err)
	}
	log.Infof("query user: %+v", queryUser)

	// Delete record
	deleteUser := User{ID: insertUser.ID}
	if err := cli.Delete(&deleteUser).Error; err != nil {
		panic(err)
	}
	log.Info("delete record succeed")

	// For more use cases, see https://gorm.io/docs/create.html
}
