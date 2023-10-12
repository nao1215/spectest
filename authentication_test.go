package spectest

import "testing"

func TestBasicAuthAuth(t *testing.T) {
	type fields struct {
		userName string
		password string
	}
	type args struct {
		userName string
		password string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "auth with valid credentials",
			fields: fields{
				userName: "user",
				password: "password",
			},
			args: args{
				userName: "user",
				password: "password",
			},
			wantErr: false,
		},
		{
			name: "auth with invalid credentials. bad password",
			fields: fields{
				userName: "user",
				password: "password",
			},
			args: args{
				userName: "user",
				password: "invalid-password",
			},
			wantErr: true,
		},
		{
			name: "auth with invalid credentials. bad user name",
			fields: fields{
				userName: "user",
				password: "password",
			},
			args: args{
				userName: "invalid-user",
				password: "password",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := &basicAuth{
				userName: tt.fields.userName,
				password: tt.fields.password,
			}
			if err := b.auth(tt.args.userName, tt.args.password); (err != nil) != tt.wantErr {
				t.Errorf("basicAuth.auth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
