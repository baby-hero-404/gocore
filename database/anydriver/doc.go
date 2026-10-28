// Package anydriver selects the database driver at runtime from database.Config.Driver.
//
// Importing it links every driver (PostgreSQL, MySQL and SQLite, which needs CGO). Services that
// know their driver at compile time should import database/postgres, database/mysql or
// database/sqlite directly and carry only that driver.
package anydriver
