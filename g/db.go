package g

import (
	"database/sql"
	"github.com/dinp/common/model"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

var DB *sql.DB

func InitDbConnPool() {
	var err error
	dbDsn := Config().DB.Dsn

	DB, err = sql.Open("mysql", dbDsn)
	if err != nil {
		log.Fatalf("sql.Open %s fail: %s", dbDsn, err)
	}

	DB.SetMaxIdleConns(Config().DB.MaxIdle)

	err = DB.Ping()
	if err != nil {
		log.Fatalf("Ping() fail: %s", err)
	}
}

func LoadEnvVarsOf(appName string) (envVars map[string]string, err error) {
	envVars = make(map[string]string)

	sq := "select k, v from env where app_name = ?"

	var stmt *sql.Stmt
	stmt, err = DB.Prepare(sq)
	if err != nil {
		log.Printf("[ERROR] prepare sql: %s fail: %v, params: [%s]", sq, err, appName)
		return
	}

	defer stmt.Close()

	var rows *sql.Rows
	rows, err = stmt.Query(appName)
	if err != nil {
		log.Printf("[ERROR] exec sql: %s fail: %v, params: [%s]", sq, err, appName)
		return
	}

	for rows.Next() {
		var k, v string
		err = rows.Scan(&k, &v)
		if err != nil {
			log.Printf("[ERROR] %s scan fail: %s", sq, err)
			return
		}

		envVars[k] = v
	}

	return
}

func UpdateAppStatus(app *model.App, status int) error {
	if Config().Debug {
		log.Printf("[INFO] udpate app: %s status to: %d", app.Name, status)
	}

	sq := "update app set status = ? where name = ?"
	stmt, err := DB.Prepare(sq)
	if err != nil {
		log.Printf("[ERROR] prepare sql: %s fail: %v, params: [%d, %s]", sq, err, status, app.Name)
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(status, app.Name)
	if err != nil {
		log.Printf("[ERROR] exec sql: %s fail: %v, params: [%d, %s]", sq, err, status, app.Name)
		return err
	}

	return nil
}
