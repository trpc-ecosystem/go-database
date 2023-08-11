package kafka

import (
	"testing"

	"github.com/Shopify/sarama"
	"github.com/golang/mock/gomock"
	"github.com/xdg-go/scram"
	"trpc.group/trpc-go/trpc-database/kafka/mockkafka"
)

func TestBegin(t *testing.T) {
	type fields struct {
		Client             *scram.Client
		ClientConversation *scram.ClientConversation
		HashGeneratorFcn   scram.HashGeneratorFcn
		User               string
		Password           string
		Mechanism          string
	}
	type args struct {
		userName string
		password string
		authzID  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"BeginFail",
			fields{
				Client:             &scram.Client{},
				ClientConversation: &scram.ClientConversation{},
				User:               "user",
				Password:           "password",
				Mechanism:          "mechanism",
			},
			args{
				"userName",
				"password",
				"authzID",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LSCRAMClient{
				Client:             tt.fields.Client,
				ClientConversation: tt.fields.ClientConversation,
				HashGeneratorFcn:   tt.fields.HashGeneratorFcn,
				User:               tt.fields.User,
				Password:           tt.fields.Password,
				Mechanism:          tt.fields.Mechanism,
			}
			if err := s.Begin(tt.args.userName, tt.args.password, tt.args.authzID); (err != nil) != tt.wantErr {
				t.Errorf("Begin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDone(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	cliCon := mockkafka.NewMockClientConversation(ctl)
	cliCon.EXPECT().Done().Return(true).AnyTimes()
	cliCon.EXPECT().Step(gomock.Any()).Return("response", nil).AnyTimes()

	type fields struct {
		Client             *scram.Client
		ClientConversation ClientConversation
		HashGeneratorFcn   scram.HashGeneratorFcn
		User               string
		Password           string
		Mechanism          string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			"DoneSucc",
			fields{
				Client: &scram.Client{},
				// ClientConversation: &scram.ClientConversation{},
				ClientConversation: cliCon,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LSCRAMClient{
				Client:             tt.fields.Client,
				ClientConversation: tt.fields.ClientConversation,
				HashGeneratorFcn:   tt.fields.HashGeneratorFcn,
				User:               tt.fields.User,
				Password:           tt.fields.Password,
				Mechanism:          tt.fields.Mechanism,
			}
			if got := s.Done(); got != tt.want {
				t.Errorf("Done() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	type fields struct {
		Client             *scram.Client
		ClientConversation *scram.ClientConversation
		HashGeneratorFcn   scram.HashGeneratorFcn
		User               string
		Password           string
		Mechanism          string
	}
	type args struct {
		vals []string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantFiled *fields
	}{
		{
			"user",
			fields{},
			args{vals: []string{"user", "userName"}},
			&fields{
				User: "userName",
			},
		},
		{
			"password",
			fields{},
			args{vals: []string{"password", "password"}},
			&fields{
				Password: "password",
			},
		},
		{
			"mechanism",
			fields{},
			args{vals: []string{"mechanism", "mechanism"}},
			&fields{
				Mechanism: "mechanism",
			},
		},
		{
			"other",
			fields{},
			args{vals: []string{"otherField", "otherValue"}},
			&fields{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LSCRAMClient{
				Client:             tt.fields.Client,
				ClientConversation: tt.fields.ClientConversation,
				HashGeneratorFcn:   tt.fields.HashGeneratorFcn,
				User:               tt.fields.User,
				Password:           tt.fields.Password,
				Mechanism:          tt.fields.Mechanism,
			}
			s.Parse(tt.args.vals)
			if s.User != tt.wantFiled.User || s.Password != tt.wantFiled.Password || s.Mechanism != tt.wantFiled.Mechanism {
				t.Errorf("test %s error, want: %+v, actual: %+v", tt.name, *tt.wantFiled, *s)
			}
		})
	}
}

func TestStep(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	cliCon := mockkafka.NewMockClientConversation(ctl)
	cliCon.EXPECT().Done().Return(true).AnyTimes()
	cliCon.EXPECT().Step(gomock.Any()).Return("response", nil).AnyTimes()

	type fields struct {
		Client             *scram.Client
		ClientConversation ClientConversation
		HashGeneratorFcn   scram.HashGeneratorFcn
		User               string
		Password           string
		Mechanism          string
	}
	type args struct {
		challenge string
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantResponse string
		wantErr      bool
	}{
		{
			"StepSucc",
			fields{
				Client:             &scram.Client{},
				ClientConversation: cliCon,
			},
			args{challenge: ""},
			"response",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LSCRAMClient{
				Client:             tt.fields.Client,
				ClientConversation: tt.fields.ClientConversation,
				HashGeneratorFcn:   tt.fields.HashGeneratorFcn,
				User:               tt.fields.User,
				Password:           tt.fields.Password,
				Mechanism:          tt.fields.Mechanism,
			}
			gotResponse, err := s.Step(tt.args.challenge)
			if (err != nil) != tt.wantErr {
				t.Errorf("Step() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotResponse != tt.wantResponse {
				t.Errorf("Step() gotResponse = %v, want %v", gotResponse, tt.wantResponse)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	type fields struct {
		Client             *scram.Client
		ClientConversation *scram.ClientConversation
		HashGeneratorFcn   scram.HashGeneratorFcn
		User               string
		Password           string
		Mechanism          string
		Protocol           string
	}
	type args struct {
		config *sarama.Config
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"MechanismEmpty",
			fields{
				Client:             &scram.Client{},
				ClientConversation: &scram.ClientConversation{},
				Mechanism:          "",
			},
			args{config: &sarama.Config{}},
			true,
		},
		{
			"MechanismSHA512",
			fields{
				Client:             &scram.Client{},
				ClientConversation: &scram.ClientConversation{},
				Mechanism:          sarama.SASLTypeSCRAMSHA512,
			},
			args{config: &sarama.Config{}},
			false,
		},
		{
			"MechanismSHA256",
			fields{
				Client:             &scram.Client{},
				ClientConversation: &scram.ClientConversation{},
				Mechanism:          sarama.SASLTypeSCRAMSHA256,
			},
			args{config: &sarama.Config{}},
			false,
		},
		{
			"MechanismSHAOther",
			fields{
				Client:             &scram.Client{},
				ClientConversation: &scram.ClientConversation{},
				Mechanism:          "123123",
			},
			args{config: &sarama.Config{}},
			true,
		},
		{
			"Protocol-SASL_SSL",
			fields{
				Client:             &scram.Client{},
				ClientConversation: &scram.ClientConversation{},
				Mechanism:          "123123",
				Protocol:           SASLTypeSSL,
			},
			args{config: &sarama.Config{}},
			true,
		},
		{
			"Protocol-default",
			fields{
				Client:             &scram.Client{},
				ClientConversation: &scram.ClientConversation{},
				Mechanism:          "123123",
				Protocol:           "123",
			},
			args{config: &sarama.Config{}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &LSCRAMClient{
				Client:             tt.fields.Client,
				ClientConversation: tt.fields.ClientConversation,
				HashGeneratorFcn:   tt.fields.HashGeneratorFcn,
				User:               tt.fields.User,
				Password:           tt.fields.Password,
				Mechanism:          tt.fields.Mechanism,
				Protocol:           tt.fields.Protocol,
			}
			if err := s.config(tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("config() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
