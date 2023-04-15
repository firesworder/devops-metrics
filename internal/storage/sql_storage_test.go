package storage

import (
	"github.com/firesworder/devopsmetrics/internal/mock_dbstore"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateTableIfNotExist(t *testing.T) {
	// определяем mock контроллер
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mock_dbstore.NewMockDBStorage(ctrl)
	dbMock.EXPECT().Exec(gomock.Any()).Return(nil, nil).Times(1)
	db := SqlStorage{Connection: dbMock}

	err := db.CreateTableIfNotExist()
	require.NoError(t, err)
}
