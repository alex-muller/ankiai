package config

func Get() Config {
	return Config{
		DBPath:         `tmp/anki_vocabulary.db`,
		MigrationsPath: `migrations`,
	}
}

type Config struct {
	DBPath         string
	MigrationsPath string
}
