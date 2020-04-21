// TODO:
// * make it work as a library along with executable
// * add couple retries while connecting to database
// * research output formats
// * prettier table create statements

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"database/sql"
	_ "github.com/ClickHouse/clickhouse-go"
)

func main() {
	var helpPtr = flag.Bool("help", false, " Print usage")
	var clickhouseUrlPtr = flag.String("url", "", " ClickHouse url with port, user and password if needed (clickhouse://your.host:9000?username=default&password=&x-multi-statement=true)")
	var filePtr = flag.String("file", "schema.sql", " Output file with path")

	flag.Parse()

	if *helpPtr || len(*clickhouseUrlPtr) < 1 {
		flag.PrintDefaults()
		os.Exit(0)
	}

	writeSchema(clickhouseUrlPtr, filePtr)
}

func writeSchema(url *string, path *string) {
	db := connectToClickHouse(url)
	defer db.Close()

	fd, err := os.OpenFile(*path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	defer fd.Close()
	if err != nil {
		log.Fatalf("opening file: %v", err)
	}

	databases := getDatabases(db)

	for _, dbName := range databases {
		if dbName == "system" {
			continue // skip system database
		}
		dbCreateStmt := dbCreateStmt(db, dbName)
		fd.Write([]byte(dbCreateStmt + "\n\n"))
		tables := getTables(db, dbName)
		for _, tableName := range tables {
			tableCreateStmt := tableCreateStmt(db, dbName, tableName)
			fd.Write([]byte(tableCreateStmt + "\n\n"))
		}
	}
}

func tableCreateStmt(db *sql.DB, dbName string, tableName string) string {
	var createStmt string
	queryStmt := fmt.Sprintf("SHOW CREATE TABLE %s.%s FORMAT PrettySpaceNoEscapes;", dbName, tableName)
	err := db.QueryRow(queryStmt).Scan(&createStmt)
	if err != nil {
		log.Fatalf("getting table %s.%s statement: %v", dbName, tableName, err)
	}

	return createStmt
}

func dbCreateStmt(db *sql.DB, dbName string) string {
	var createStmt string
	queryStmt := fmt.Sprintf("SHOW CREATE DATABASE %s FORMAT TabSeparated;", dbName)
	err := db.QueryRow(queryStmt).Scan(&createStmt)
	if err != nil {
		log.Fatalf("getting database %s statement: %v", dbName, err)
	}

	return createStmt
}

func getTables(db *sql.DB, dbName string) []string {
	var tables []string
	rows, err := db.Query("SELECT name FROM system.tables WHERE database = ?;", dbName)
	if err != nil {
		log.Fatalf("getting tables for %s: %v", dbName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatalf("getting tables for %s: %v", dbName, err)
		}
		tables = append(tables, name)
	}

	if rows.Err(); err != nil {
		log.Fatalf("getting tables for %s: %v", dbName, err)
	}

	return tables
}

func getDatabases(db *sql.DB) []string {
	var databases []string
	rows, err := db.Query("SHOW DATABASES FORMAT TabSeparated;")
	if err != nil {
		log.Fatalf("getting databases: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatalf("getting databases: %v", err)
		}
		databases = append(databases, name)
	}

	if rows.Err(); err != nil {
		log.Fatalf("getting databases: %v", err)
	}

	return databases
}

func connectToClickHouse(url *string) *sql.DB {
	db, err := sql.Open("clickhouse", *url)
	if err != nil {
		log.Fatalf("getting clickhouse connection: %v", err)
	}

	if _, err := db.Exec("SELECT 1"); err != nil {
		log.Fatalf("trying to ping clickhouse: %v", err)
	}

	return db
}