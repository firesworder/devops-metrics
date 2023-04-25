package message

import (
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMetrics_InitHash(t *testing.T) {
	fl64, i64 := float64(12.3), int64(12)

	type args struct {
		key string
	}
	tests := []struct {
		name     string
		msg      Metrics
		args     args
		wantHash string
		wantErr  bool
	}{
		{
			name: "Test 1. Correct obj, gauge type and key are set.",
			msg: Metrics{
				ID:    "RandomValue",
				MType: internal.GaugeTypeName,
				Delta: nil,
				Value: &fl64,
				Hash:  "",
			},
			args:     args{key: "Ayayaka"},
			wantHash: "b375cfb09d5bacee9632cf2305b92c6fbb08fba491bca87392bde97d7a137961",
			wantErr:  false,
		},
		{
			name: "Test 2. Correct obj, counter type and key are set.",
			msg: Metrics{
				ID:    "PollCount",
				MType: internal.CounterTypeName,
				Delta: &i64,
				Value: nil,
				Hash:  "",
			},
			args:     args{key: "Ayayaka"},
			wantHash: "0246e9b1bd13ab3a2b3f44c7c6ae4f286b1855fd3f4fc46d0dc2707dcff22381",
			wantErr:  false,
		},
		{
			name: "Test 3. Correct obj, any metric type. Key is not set.",
			msg: Metrics{
				ID:    "PollCount",
				MType: internal.CounterTypeName,
				Delta: &i64,
				Value: nil,
				Hash:  "",
			},
			args:     args{key: ""},
			wantHash: "",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.InitHash(tt.args.key)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantHash, tt.msg.Hash)
		})
	}
}
