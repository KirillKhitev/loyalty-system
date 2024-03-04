package auth

import (
	"testing"
)

func TestAuthorizingData_GenerateHashPassword(t *testing.T) {
	type fields struct {
		Login    string
		Password string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "#1 positive",
			fields: fields{
				Login:    "Maksim",
				Password: "54453535trgg345",
			},
			want: "8bf8322ded60dd9185198970a2f02e35a34bfdd6467f71a0602ce54f29114d29",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &AuthorizingData{
				Login:    tt.fields.Login,
				Password: tt.fields.Password,
			}
			if got := d.GenerateHashPassword(); got != tt.want {
				t.Errorf("GenerateHashPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUserIDFromAuthHeader(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			name:    "1 positive",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := AuthorizingData{
				Login:    "test",
				Password: "passwordUser",
			}

			user := data.NewUserFromData()

			token, err := BuildJWTString(user)
			if err != nil {
				t.Errorf("BuildJWTString() error = %v", err)
				return
			}

			header := "Bearer " + token
			got, err := GetUserIDFromAuthHeader(header)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserIDFromAuthHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != user.ID {
				t.Errorf("GetUserIDFromAuthHeader() got = %v, want %v", got, tt.want)
			}
		})
	}
}
