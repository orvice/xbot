package conf

type Config struct{}

var (
	Conf = new(Config)
)

func (c *Config) Print() {}
