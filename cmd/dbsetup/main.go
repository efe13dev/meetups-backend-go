package main

import (
	"database/sql"
	"fmt"
	"log"
	"meetups-backend-go/internal/db"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	db.LoadEnv(".env")
	dbName := os.Getenv("DATABASE_NAME")
	if dbName == "" {
		log.Fatal("DATABASE_NAME no está definido en .env")
	}

	cmd := "all"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	pool, err := db.Open(false)
	if err != nil {
		log.Fatalf("Error conectando a MySQL: %v", err)
	}
	defer pool.Close()

	switch cmd {
	case "ping":
		fmt.Println("MySQL connection OK!")
	case "init":
		if err := initDB(pool, dbName); err != nil {
			log.Fatal(err)
		}
	case "populate":
		if err := populateDB(pool, dbName); err != nil {
			log.Fatal(err)
		}
	case "photos":
		if err := updatePhotos(pool, dbName); err != nil {
			log.Fatal(err)
		}
	case "all":
		if err := initDB(pool, dbName); err != nil {
			log.Fatal(err)
		}
		if err := populateDB(pool, dbName); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("Uso: go run ./cmd/dbsetup [init|populate|all]")
	}
}

func initDB(pool *sql.DB, dbName string) error {
	fmt.Println("Recreating database and tables...")
	// Un único viaje a MySQL gracias a multiStatements=true
	schema := fmt.Sprintf(`
DROP DATABASE IF EXISTS %[1]s;
CREATE DATABASE %[1]s;
USE %[1]s;

CREATE TABLE users (
	id INT UNSIGNED UNIQUE AUTO_INCREMENT NOT NULL PRIMARY KEY,
	email VARCHAR(50) NOT NULL UNIQUE,
	password VARCHAR(100) NOT NULL,
	name VARCHAR(50) NOT NULL,
	biography VARCHAR(2000),
	avatar VARCHAR(500) DEFAULT 'avatar-default.png'
);

CREATE TABLE meetups (
	id INT UNSIGNED AUTO_INCREMENT NOT NULL PRIMARY KEY,
	title VARCHAR(100) NOT NULL,
	description VARCHAR(4000) NOT NULL,
	photo VARCHAR(500) DEFAULT 'default-meetup-image.png',
	category ENUM('Fotografia','Ciencia','Tecnologia','Gastronomia','Naturaleza','Literatura','Musica','Salud','Arte','Cultura','Otras'),
	city ENUM('Andalucia','Aragon','Asturias','Cantabria','Castilla_León','Castilla_La_Mancha','Cataluña','Madrid','Valencia','Extremadura','Galicia','Baleares','Canarias','Rioja','Murcia','Navarra','Pais_Vasco'),
	date DATETIME NOT NULL,
	user_id INT UNSIGNED NOT NULL,
	INDEX idx_meetups_date (date),
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE inscriptions (
	id_user INT UNSIGNED NOT NULL,
	id_meetup INT UNSIGNED NOT NULL,
	PRIMARY KEY (id_user, id_meetup),
	INDEX idx_inscriptions_meetup (id_meetup),
	FOREIGN KEY (id_user) REFERENCES users (id) ON DELETE CASCADE,
	FOREIGN KEY (id_meetup) REFERENCES meetups (id) ON DELETE CASCADE
);`, dbName)

	if _, err := pool.Exec(schema); err != nil {
		return err
	}
	fmt.Println("Database and tables created.")
	return nil
}

func populateDB(pool *sql.DB, dbName string) error {
	fmt.Println("Populating database...")
	// Hasheamos '12345' una sola vez en lugar de 5 veces seguidas como hacía Node
	hashBytes, err := bcrypt.GenerateFromPassword([]byte("12345"), 10)
	if err != nil {
		return err
	}
	pass := string(hashBytes)

	seed := fmt.Sprintf("USE %s;\n", dbName) + `
INSERT INTO users (email, password, name, biography, avatar) VALUES
('julia@email.com', ?, 'Julia', 'Soy Julia, una incansable exploradora, que ha recorrido selvas y montañas en busca de la naturaleza en su forma más pura.', 'a6945dca-9aa9-414b-ad39-d09dfece78e9.jpeg'),
('pepe@email.com', ?, 'Pepe', 'Pepe, apasionado de la música desde niño, se convirtió en un virtuoso del piano a temprana edad.', '0353105b-249d-4908-8472-ceaf703063cf.jpeg'),
('maria@email.com', ?, 'Maria', 'María, la panadera de corazón, dedicó su vida al arte del pan.', '49fc3e52-336e-48df-aea6-598751a1e498.jpeg'),
('juan@email.com', ?, 'Juan', 'Juan, amante de la sencillez, disfruta de la vida en dos ruedas.', 'd3ff7899-4eb2-4aa4-911d-721f52bdd76d.jpeg'),
('jorge@email.com', ?, 'Jorge', 'Jorge, el visionario de la tecnología, fundó empresas innovadoras que revolucionaron la industria.', '130450a0-edd6-4bb1-ac52-f7820023d3f8.jpeg');

INSERT INTO meetups (title, description, photo, category, city, date, user_id) VALUES
('Taller de Fotografía Nocturna', 'Aprende las técnicas de fotografía nocturna con expertos en el campo.', '808d7b69-e91e-424a-8bec-cfde9a9a3ee9.jpeg', 'Fotografia', 'Asturias', DATE_ADD(NOW(), INTERVAL 3 DAY), 1),
('Charla sobre Inteligencia Artificial en la Medicina', 'Explora el impacto de la IA en la atención médica.', '6fb161d6-01a2-47d1-9e32-73e0bc88cc15.jpeg', 'Ciencia', 'Cataluña', DATE_ADD(NOW(), INTERVAL 5 DAY), 5),
('Networking para Emprendedores', 'Conecta con otros emprendedores y comparte experiencias.', 'a1abce3a-422f-4c57-a323-af024ab8fd36.jpeg', 'Tecnologia', 'Madrid', DATE_ADD(NOW(), INTERVAL 7 DAY), 2),
('Curso de Yoga al Aire Libre', 'Encuentra paz y equilibrio practicando yoga en la naturaleza.', '9606860d-776e-4045-96d1-b317560ab4b4.jpeg', 'Salud', 'Baleares', DATE_ADD(NOW(), INTERVAL 10 DAY), 3),
('Taller de Cocina Vegetariana', 'Aprende a preparar deliciosos platos vegetarianos.', '1f08a4fd-d7d2-42cf-bf67-70c9e9f7fc63.jpeg', 'Gastronomia', 'Murcia', DATE_ADD(NOW(), INTERVAL 12 DAY), 3),
('Conferencia de Cambio Climático', 'Explora soluciones para combatir el cambio climático.', '1121d55c-db32-463d-bbf3-6b5c51de6c25.jpeg', 'Ciencia', 'Valencia', DATE_ADD(NOW(), INTERVAL 14 DAY), 5),
('Grupo de Lectura: Libro del Mes', 'Un espacio para discutir y compartir tus lecturas favoritas.', '54ae9184-05e2-4209-9f6a-17d611668443.jpeg', 'Literatura', 'Andalucia', DATE_ADD(NOW(), INTERVAL 16 DAY), 4),
('Taller de Desarrollo de Videojuegos', 'Aprende a crear tus propios videojuegos desde cero.', '3a24b0c2-d9f2-4be9-9d0e-573522833657.jpeg', 'Tecnologia', 'Extremadura', DATE_ADD(NOW(), INTERVAL 20 DAY), 5);

INSERT INTO inscriptions (id_user, id_meetup) VALUES
(1, 2), (1, 5), (2, 2), (3, 2), (4, 2);`

	if _, err := pool.Exec(seed, pass, pass, pass, pass, pass); err != nil {
		return err
	}
	fmt.Println("Database populated.")
	return nil
}

func updatePhotos(pool *sql.DB, dbName string) error {
	q := fmt.Sprintf("USE %s;\n", dbName) + `
UPDATE meetups SET photo = '808d7b69-e91e-424a-8bec-cfde9a9a3ee9.jpeg' WHERE id = 1;
UPDATE meetups SET photo = '6fb161d6-01a2-47d1-9e32-73e0bc88cc15.jpeg' WHERE id = 2;
UPDATE meetups SET photo = 'a1abce3a-422f-4c57-a323-af024ab8fd36.jpeg' WHERE id = 3;
UPDATE meetups SET photo = '9606860d-776e-4045-96d1-b317560ab4b4.jpeg' WHERE id = 4;
UPDATE meetups SET photo = '1f08a4fd-d7d2-42cf-bf67-70c9e9f7fc63.jpeg' WHERE id = 5;
UPDATE meetups SET photo = '1121d55c-db32-463d-bbf3-6b5c51de6c25.jpeg' WHERE id = 6;
UPDATE meetups SET photo = '54ae9184-05e2-4209-9f6a-17d611668443.jpeg' WHERE id = 7;
UPDATE meetups SET photo = '3a24b0c2-d9f2-4be9-9d0e-573522833657.jpeg' WHERE id = 8;

UPDATE users SET avatar = 'a6945dca-9aa9-414b-ad39-d09dfece78e9.jpeg' WHERE id = 1;
UPDATE users SET avatar = '0353105b-249d-4908-8472-ceaf703063cf.jpeg' WHERE id = 2;
UPDATE users SET avatar = '49fc3e52-336e-48df-aea6-598751a1e498.jpeg' WHERE id = 3;
UPDATE users SET avatar = 'd3ff7899-4eb2-4aa4-911d-721f52bdd76d.jpeg' WHERE id = 4;
UPDATE users SET avatar = '130450a0-edd6-4bb1-ac52-f7820023d3f8.jpeg' WHERE id = 5;`
	_, err := pool.Exec(q)
	if err == nil {
		fmt.Println("Photos and avatars updated.")
	}
	return err
}
