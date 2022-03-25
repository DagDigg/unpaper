package helpers_test

import (
	"testing"

	"github.com/DagDigg/unpaper/backend/helpers"
	"github.com/stretchr/testify/assert"
)

func TestIsValidEmail(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	tests := []struct {
		mail string
		want bool
	}{
		{
			mail: "works@gmail.com",
			want: true,
		},
		{
			mail: "invalid",
			want: false,
		},
		{
			mail: "invalid@inv",
			want: false,
		},
		{
			mail: "invalid@shouldntwork.it",
			want: false,
		},
		{
			mail: "works@me.com",
			want: true,
		},
		{
			mail: "works@yahoo.com",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.mail, func(t *testing.T) {
			t.Parallel()
			got, _ := helpers.IsEmailValid(tt.mail)
			assert.Equal(got, tt.want)
		})
	}
}
