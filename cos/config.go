package cos

// Conf is the configuration information for cos server.
type Conf struct {
	AppID string `yaml:"app_id"` // AppID
	// the bucket name, corresponding to BucketName in Tencent Cloud cos, not APPID+BucketName
	Bucket string `yaml:"bucket"`
	// the ssm tag, used to get secret_id and secret_key, optional parameter
	SsmTag       string `yaml:"ssm_tag"`
	SecretID     string `yaml:"secret_id"`
	SecretKey    string `yaml:"secret_key"`
	Region       string `yaml:"region"`
	Domain       string `yaml:"domain"`
	Scheme       string `yaml:"scheme"`       // the protocol, default https
	SessionToken string `yaml:"sessiontoken"` // the temporary secret key, optional parameter
	// the signature expiration time, in seconds, optional parameter, default 1800s if not specified
	SignExpiration int64 `yaml:"sign_expiration"`
	// the domain name prefix, optional parameter, default cos, can be customized
	Prefix  string `yaml:"prefix"`
	BaseURL string `yaml:"base_url"` // BaseURL follows the standard COS configuration format
}
