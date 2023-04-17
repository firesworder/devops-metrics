package storage

import (
	"errors"
	"github.com/firesworder/devopsmetrics/internal/mock_dbstore"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var mockError = errors.New("error")

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

func TestSqlStorage_AddMetric(t *testing.T) {
	// определяем mock контроллер
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// mock db
	dbMock := mock_dbstore.NewMockDBStorage(ctrl)
	db := SqlStorage{Connection: dbMock}

	// mock stmt
	execInput := metric1Counter10.Name
	stmtMock := mock_dbstore.NewMockDBStmt(ctrl)
	stmtMock.EXPECT().Exec(execInput).Return(nil, nil).Times(1)
	stmtMock.EXPECT().Exec(execInput).Return(nil, mockError).Times(1)
	insertStmt = stmtMock

	tests := []struct {
		name string
		Metric
		wantErr bool
	}{
		{
			name:    "Test 1. Metric not present in db.",
			Metric:  metric1Counter10,
			wantErr: false,
		},
		{
			name:    "Test 2. Metric present in db.",
			Metric:  metric1Counter10,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.AddMetric(tt.Metric)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestSqlStorage_DeleteMetric(t *testing.T) {
	// определяем mock контроллер
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// mock db
	dbMock := mock_dbstore.NewMockDBStorage(ctrl)
	db := SqlStorage{Connection: dbMock}

	// mock sql.result
	resultMock := mock_dbstore.NewMockDBResult(ctrl)
	resultMock.EXPECT().RowsAffected().Return(int64(1), nil).Times(1)

	// mock stmt
	execInputT1, execInputT2 := metric7Counter27.Name, metric1Counter10.Name
	stmtMock := mock_dbstore.NewMockDBStmt(ctrl)
	stmtMock.EXPECT().Exec(execInputT1).Return(nil, mockError).Times(1)
	stmtMock.EXPECT().Exec(execInputT2).Return(resultMock, nil).Times(1)
	deleteStmt = stmtMock

	tests := []struct {
		name string
		Metric
		wantErr bool
	}{
		{
			name:    "Test 1. Metric not present in db.",
			Metric:  metric7Counter27,
			wantErr: true,
		},
		{
			name:    "Test 2. Metric present in db.",
			Metric:  metric1Counter10,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.DeleteMetric(tt.Metric)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestSqlStorage_IsMetricInStorage(t *testing.T) {
	// определяем mock контроллер
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// mock db
	dbMock := mock_dbstore.NewMockDBStorage(ctrl)
	db := SqlStorage{Connection: dbMock}

	// mock sql.result
	resultMock := mock_dbstore.NewMockDBResult(ctrl)
	resultMock.EXPECT().RowsAffected().Return(int64(0), nil).Times(1)
	resultMock.EXPECT().RowsAffected().Return(int64(1), nil).Times(1)

	// mock stmt
	execInputT1, execInputT2 := metric7Counter27.Name, metric1Counter10.Name
	stmtMock := mock_dbstore.NewMockDBStmt(ctrl)
	stmtMock.EXPECT().Exec(execInputT1).Return(resultMock, nil).Times(1)
	stmtMock.EXPECT().Exec(execInputT2).Return(resultMock, nil).Times(1)
	selectMetricStmt = stmtMock

	tests := []struct {
		name string
		Metric
		wantResult bool
	}{
		{
			name:       "Test 1. Metric not present in db.",
			Metric:     metric7Counter27,
			wantResult: false,
		},
		{
			name:       "Test 2. Metric present in db.",
			Metric:     metric1Counter10,
			wantResult: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult := db.IsMetricInStorage(tt.Metric)
			assert.Equal(t, tt.wantResult, gotResult)
		})
	}
}

func TestSqlStorage_UpdateOrAddMetric(t *testing.T) {
	// определяем mock контроллер
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// mock db
	dbMock := mock_dbstore.NewMockDBStorage(ctrl)
	db := SqlStorage{Connection: dbMock}

	// mock sql.result
	resultMock := mock_dbstore.NewMockDBResult(ctrl)
	// test1: для IsMetricInStorage
	resultMock.EXPECT().RowsAffected().Return(int64(0), nil).Times(1)
	// test2: для IsMetricInStorage и Update
	resultMock.EXPECT().RowsAffected().Return(int64(1), nil).Times(2)

	// mock stmt
	stmtMockSelect := mock_dbstore.NewMockDBStmt(ctrl)
	stmtMockSelect.EXPECT().Exec(metric7Counter27.Name).Return(resultMock, nil).Times(1)
	stmtMockSelect.EXPECT().Exec(metric1Counter10.Name).Return(resultMock, nil).Times(1)
	selectMetricStmt = stmtMockSelect

	stmtMockAdd := mock_dbstore.NewMockDBStmt(ctrl)
	// test1
	stmtMockAdd.EXPECT().Exec(metric7Counter27.GetMetricParamsString()).
		Return(nil, nil).Times(1)
	// test2
	stmtMockUpdate := mock_dbstore.NewMockDBStmt(ctrl)
	stmtMockUpdate.EXPECT().Exec(metric1Counter10.GetMetricParamsString()).
		Return(resultMock, nil).Times(1)
	// установка моков стейтмента
	insertStmt, updateStmt = stmtMockAdd, stmtMockUpdate

	tests := []struct {
		name string
		Metric
		wantError error
	}{
		{
			name:      "Test 1. Metric not present in db. Add.",
			Metric:    metric7Counter27,
			wantError: nil,
		},
		{
			name:      "Test 2. Metric present in db. Update.",
			Metric:    metric1Counter10,
			wantError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.UpdateOrAddMetric(tt.Metric)
			assert.Equal(t, tt.wantError, err)
		})
	}
}
