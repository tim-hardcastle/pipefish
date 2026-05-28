package hub

// When I do the SQL integration I can just put this stuff in the hub, replacing it with more generic
// methods for interacting with Pipefish code. Hence, no efforts to keep it DRY and minimal efforts at
// error-handling.
//
// TODO --- you can in fact do that now.

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"math/big"
	"sort"

	"golang.org/x/crypto/bcrypt"

	// SQL drivers

	_ "github.com/databricks/databricks-sql-go" // Databricks
	_ "github.com/go-sql-driver/mysql"          // MariaDB, MySQL, TiDB
	_ "github.com/lib/pq"                       // PostgreSQL, CockroachDB
	_ "github.com/microsoft/go-mssqldb"         // Microsoft SQL Server
	_ "github.com/nakagami/firebirdsql"         // Firebird
	_ "github.com/sijms/go-ora"                 // Oracle
	_ "modernc.org/sqlite"                      // SQLite
)

// List of SQL drivers for when I want to import more: https://zchee.github.io/golang-wiki/SQLDrivers/

func AddAdmin(db *sql.DB, username, firstName, lastName, email, password string) error {

	query :=
		`CREATE TABLE IF NOT EXISTS PipefishUsers (
    username varchar(32),
    firstName varchar(32),
    lastName varchar(32),
    password varchar(60),
    email varchar(60),
PRIMARY KEY (username));

CREATE TABLE IF NOT EXISTS PipefishGroups (
    groupName varchar(32),
PRIMARY KEY (groupName));

CREATE TABLE IF NOT EXISTS PipefishGroupMemberships (
    username varchar(32) REFERENCES PipefishUsers ON DELETE CASCADE,
    groupName varchar(32) REFERENCES PipefishGroups ON DELETE CASCADE,
    owner BOOLEAN DEFAULT FALSE,
PRIMARY KEY (username, groupName));

CREATE TABLE IF NOT EXISTS PipefishGroupServices (
    groupName varchar(32) REFERENCES PipefishGroups ON DELETE CASCADE,
	serviceName varchar(32),
PRIMARY KEY (groupName, serviceName));

INSERT INTO PipefishGroups (groupName)
VALUES('Admin')
ON CONFLICT DO NOTHING;

INSERT INTO PipefishGroups (groupName)
VALUES('Users')
ON CONFLICT DO NOTHING;

INSERT INTO PipefishGroups (groupName)
VALUES('Guests')
ON CONFLICT DO NOTHING;`
	_, err := db.Exec(query)
	if err != nil {
		return err
	}

	err = AddUser(db, username, firstName, lastName, email, password)
	if err != nil {
		return err
	}
	for _, group := range []string{"Admin", "Guests", "Users"} {
		err = AddUserToGroup(db, username, group, true)
		if err != nil {
			return err
		}
	}
	return err
}

func AddUserToGroup(db *sql.DB, username, groupName string, owner bool) error {
	query :=
		`INSERT INTO PipefishGroupMemberships(username, groupName, owner)
	VALUES ($1, $2, $3)`
	_, err := db.Exec(query, username, groupName, owner)
	return err
}

func ChangePassword(db *sql.DB, username, newPassword string) error {
	query := `UPDATE PipefishUsers
SET password = $1
WHERE username = $2;`
	_, err := db.Exec(query, encrypt(newPassword), username)
	return err
}

func UnAddUserToGroup(db *sql.DB, username, groupName string) error {
	query :=
		`DELETE FROM PipefishGroupMemberships WHERE username = $1 AND groupName = $2`
	_, err := db.Exec(query, username, groupName)
	return err
}

func LetGroupUseService(db *sql.DB, groupName, serviceName string) error {
	query :=
		`INSERT INTO PipefishGroupServices(groupName, serviceName)
	VALUES ($1, $2)`
	_, err := db.Exec(query, groupName, serviceName)
	return err
}

func UnLetGroupUseService(db *sql.DB, groupName, serviceName string) error {
	query :=
		`DELETE FROM PipefishGroupServices WHERE groupName = $1 AND serviceName = $2`
	_, err := db.Exec(query, groupName, serviceName)
	return err
}

func SetOwnership(db *sql.DB, username, groupName string, owner bool) error {
	query := `UPDATE PipefishGroupMemberships
SET owner = $3
WHERE username = $1 AND groupName = $2;`
	_, err := db.Exec(query, username, groupName, owner)
	return err
}

type groupRow struct {
	username  string
	groupName string
	owner     bool
}

func GetGroupsOfUser(db *sql.DB, username string, ownGroups bool) (string, error) {
	rows, err := db.Query("SELECT * FROM PipefishGroupMemberships WHERE username = $1", username)
	if err != nil {

		return "", err
	}
	defer rows.Close()

	var groups []groupRow

	for rows.Next() {
		var group groupRow
		if err := rows.Scan(&group.username, &group.groupName, &group.owner); err != nil {
			return "", err
		}
		groups = append(groups, group)
	}

	if len(groups) == 0 {
		if ownGroups {
			return "You are not a member of any groups.", nil
		} else {
			return "<C>" + username + "</> is not a member of any groups.", nil
		}
	}

	result := ""

	ownerGroups := []string{}
	userGroups := []string{}
	for _, v := range groups {
		if v.owner {
			ownerGroups = append(ownerGroups, v.groupName)
		} else {
			userGroups = append(userGroups, v.groupName)
		}
	}
	if len(ownerGroups) > 0 {
		sort.Strings(ownerGroups)
		if ownGroups {
			result = result + "You are an owner of the following groups:\n\n"
		} else {
			result = result + "The user <C>" + username + "</> is an owner of the following groups:\n\n"
		}
		for _, v := range ownerGroups {
			result = result + "- " + v + "\n"
		}
	}
	if len(ownerGroups) > 0 && len(userGroups) > 0 {
		result = result + "\n"
	}
	if len(userGroups) > 0 {
		sort.Strings(userGroups)
		if ownGroups {
			result = result + "You are an member of the following groups:\n\n"
		} else {
			result = result + "The user <C>" + username + "</> is a member of the following groups:\n\n"
		}
		for _, v := range userGroups {
			result = result + "- " + v + "\n"
		}
	}

	return result, nil
}

func GetUsersOfGroup(db *sql.DB, groupName string) (string, error) {
	rows, err := db.Query("SELECT * FROM PipefishGroupMemberships WHERE groupName = $1", groupName)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var users []groupRow

	for rows.Next() {
		var user groupRow
		if err := rows.Scan(&user.username, &user.groupName, &user.owner); err != nil {
			return "", err
		}
		users = append(users, user)
	}

	if len(users) == 0 {
		return "The group <C>" + groupName + "</> has no users.", nil
	}

	result := ""

	owners := []string{}
	usersOnly := []string{}
	for _, v := range users {
		if v.owner {
			owners = append(owners, v.username)
		} else {
			usersOnly = append(usersOnly, v.username)
		}
	}

	if len(owners) > 0 {
		sort.Strings(owners)
		result = result + "The group <C>" + groupName + "</> has the following owners:\n\n"
		for _, v := range owners {
			result = result + "- " + v + "\n"
		}
	}
	if len(owners) > 0 && len(usersOnly) > 0 {
		result = result + "\n"
	}
	if len(usersOnly) > 0 {
		sort.Strings(usersOnly)
		result = result + "The group <C>" + groupName + "</> has the following users:\n\n"
		for _, v := range usersOnly {
			result = result + "- " + v + "\n"
		}
	}

	return result, nil
}

func userHasService(db *sql.DB, username, servicename string) bool {
	var exists int
	err := db.QueryRow(
		`SELECT 1
FROM PipefishGroupMemberships 
INNER JOIN PipefishGroupServices
ON PipefishGroupMemberships.groupName = PipefishGroupServices.groupName
WHERE PipefishGroupMemberships.username = $1 AND PipefishGroupServices.servicename = $2
LIMIT 1`, username, servicename).Scan(&exists)
	return err == nil
}

func GetServicesOfUser(db *sql.DB, username string, ownServices bool) (string, error) {
	rows, err := db.Query(
		`SELECT PipefishGroupServices.serviceName FROM PipefishGroupMemberships 
INNER JOIN PipefishGroupServices
ON PipefishGroupMemberships.groupName = PipefishGroupServices.groupName
WHERE username = $1`, username)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var services []string

	for rows.Next() {
		var service string
		if err := rows.Scan(&service); err != nil {
			return "", err
		}
		services = append(services, service)
	}

	if len(services) == 0 {
		if ownServices {
			return "\nYou do not have access to any services.\n\n", nil
		} else {
			return "The user <C>" + username + "</> does not have access to any services.\n\n", nil
		}
	}

	result := ""

	sort.Strings(services)
	if ownServices {
		result = result + "You have access to the following services:\n\n"
	} else {
		result = result + "The user <C>" + username + "</> has access to the following services:\n\n"
	}
	for _, v := range services {
		if v != "" {
			result = result + "- " + v + "\n"
		}
	}

	return result + "\n", nil
}

func GetUsersOfService(db *sql.DB, serviceName string) (string, error) {
	rows, err := db.Query(
		`SELECT PipefishGroupMemberships.username FROM PipefishGroupServices 
INNER JOIN PipefishGroupMemberships
ON PipefishGroupMemberships.groupName = PipefishGroupServices.groupName
WHERE serviceName = $1`, serviceName)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var users []string

	for rows.Next() {
		var user string
		if err := rows.Scan(&user); err != nil {
			return "", err
		}
		users = append(users, user)
	}

	if len(users) == 0 {
		return "The service <C>" + serviceName + "</> does not have any users.\n\n", nil
	}

	result := ""

	sort.Strings(users)
	result = result + "The service <C>" + serviceName + "</> has the following users:\n\n"
	for _, v := range users {
		if v != "" {
			result = result + "- " + v + "\n"
		}
	}

	return result + "\n", nil
}

func GetServicesOfGroup(db *sql.DB, groupName string) (string, error) {
	rows, err := db.Query("SELECT serviceName FROM PipefishGroupServices WHERE groupName = $1", groupName)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var services []string

	for rows.Next() {
		var srv string
		if err := rows.Scan(&srv); err != nil {
			return "", err
		}
		services = append(services, srv)
	}

	if len(services) == 0 {
		return "The group <C>" + groupName + "</> has access to no services.", nil
	}

	result := ""

	sort.Strings(services)
	result = result + "The group <C>" + groupName + "</> has access to the following services:\n\n"
	for _, v := range services {
		result = result + "- " + v + "\n"
	}
	return result, nil
}

func GetGroupsOfService(db *sql.DB, serviceName string) (string, error) {
	rows, err := db.Query("SELECT groupName FROM PipefishGroupServices WHERE serviceName = $1", serviceName)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var groups []string

	for rows.Next() {
		var grp string
		if err := rows.Scan(&grp); err != nil {
			return "", err
		}
		groups = append(groups, grp)
	}

	if len(groups) == 0 {
		return "The service <C>" + serviceName + "</> has no groups that can access it.", nil
	}

	sort.Strings(groups)
	result := "The service <C>" + serviceName + "</> can be accessed by the following groups:\n\n"
	for _, v := range groups {
		result = result + "- " + v + "\n"
	}

	return result, nil
}

func IsUserGroupOwner(db *sql.DB, username, groupName string) error {
	var count int

	row := db.QueryRow("SELECT COUNT (*) FROM PipefishGroupMemberships WHERE username = $1 AND groupName = $2 AND owner = TRUE",
		username, groupName)
	err := row.Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("you aren't an owner of a group '" + groupName + "'.")
	}
	return nil
}

func IsUserAdmin(db *sql.DB, username string) (bool, error) {
	return IsUserInGroup(db, username, "Admin")
}

func IsUserInGroup(db *sql.DB, username, groupName string) (bool, error) {
	var count int

	row := db.QueryRow("SELECT COUNT (*) FROM PipefishGroupMemberships WHERE username = $1 AND groupName = $2",
		username, groupName)
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

func UnRegisterUser(db *sql.DB, username string) error {
	query :=
`DELETE FROM PipefishUsers
WHERE username = $1`
	_, err := db.Exec(query, username)

	return err
}

type userRow struct {
	password string
}

func ValidateUser(db *sql.DB, username, password string) error {
	var userData userRow
	row := db.QueryRow("SELECT password FROM PipefishUsers WHERE username = $1", username)
	if err := row.Scan(&userData.password); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("the hub doesn't recognize that combination of username and password")
		}
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userData.password), []byte(password)); err != nil {
		return errors.New("the hub doesn't recognize that combination of username and password")
	}
	return nil
}

type emailRow struct {
	email string
}

func ValidateEmail(db *sql.DB, username, email string) error {
	var userData emailRow
	row := db.QueryRow("SELECT email FROM PipefishUsers WHERE username = $1", username)
	if err := row.Scan(&userData.email); err != nil {
		if err == sql.ErrNoRows {
			return errors.New("the hub doesn't recognize that combination of username and email address")
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(userData.email), []byte(email)); err != nil {
		return errors.New("the hub doesn't recognize that combination of username and email address")
	}
	return nil
}

func AddUser(db *sql.DB, username, firstName, lastName, email, password string) error {
	query :=
		`INSERT INTO PipefishUsers(username, firstName, lastName, password, email)
	VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(query, username, firstName, lastName, encrypt(password), encrypt(email))

	return err
}

func CreateGroup(db *sql.DB, groupName string) error {
	query :=
		`INSERT INTO PipefishGroups(groupName)
VALUES ($1)`
	_, err := db.Exec(query, groupName)

	return err
}

func UncreateGroup(db *sql.DB, groupName string) error {
	query :=
		`DELETE FROM PipefishGroups
WHERE groupName = $1`
	_, err := db.Exec(query, groupName)

	return err
}

func DropTables(db *sql.DB) {
	query :=
		`DROP TABLE PipefishGroupServices;
DROP TABLE PipefishGroupMemberships;
DROP TABLE PipefishGroups;
DROP TABLE PipefishUsers`
	db.Exec(query)
}

func encrypt(s string) string {
	result, _ := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	return string(result)
}

func MakePassword() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, 16)
	for i := range password {
		index, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		password[i] = chars[index.Int64()]
	}
	return string(password)
}
