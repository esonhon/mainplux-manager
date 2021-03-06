package cassandra

import (
	"fmt"

	"github.com/gocql/gocql"
	"github.com/mainflux/manager"
)

var _ manager.ClientRepository = (*clientRepository)(nil)

type clientRepository struct {
	session *gocql.Session
}

// NewClientRepository instantiates Cassandra client repository.
func NewClientRepository(session *gocql.Session) manager.ClientRepository {
	return &clientRepository{session}
}

func (repo *clientRepository) Save(client manager.Client) (string, error) {
	cql := `INSERT INTO clients_by_user
		(user, id, type, name, description, access_key, meta)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	id := gocql.TimeUUID()

	if err := repo.session.Query(cql, client.Owner, id, client.Type,
		client.Name, client.Description, client.Key, client.Meta).Exec(); err != nil {
		return "", err
	}

	return id.String(), nil
}

func (repo *clientRepository) One(owner string, id string) (manager.Client, error) {
	cql := `SELECT type, name, description, access_key, meta
		FROM clients_by_user
		WHERE user = ? AND id = ? LIMIT 1`

	cli := manager.Client{}

	if err := repo.session.Query(cql, owner, id).
		Scan(&cli.Type, &cli.Name, &cli.Description, &cli.Key, &cli.Meta); err != nil {
		fmt.Println(err)
		return cli, manager.ErrNotFound
	}

	cli.Owner = owner
	cli.ID = id
	return cli, nil
}

func (repo *clientRepository) Remove(owner string, id string) error {
	cql := `DELETE FROM clients_by_user WHERE user = ? AND id = ?`
	return repo.session.Query(cql, owner, id).Exec()
}
