package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alexferl/zerohttp/pagination"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/alexferl/zerohttp-example/models"
)

// MongoStore implements Store using MongoDB.
type MongoStore struct {
	client  *mongo.Client
	db      *mongo.Database
	users   *mongo.Collection
	records *mongo.Collection
	orders  *mongo.Collection
}

// NewMongoStore creates a new MongoDB store
func NewMongoStore(uri, dbName string) (*MongoStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(dbName)
	store := &MongoStore{
		client:  client,
		db:      db,
		users:   db.Collection("users"),
		records: db.Collection("records"),
		orders:  db.Collection("orders"),
	}

	if err := store.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return store, nil
}

// Ping checks if MongoDB is reachable
func (s *MongoStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx, nil)
}

func (s *MongoStore) createIndexes(ctx context.Context) error {
	// Users indexes
	userIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}
	if _, err := s.users.Indexes().CreateMany(ctx, userIndexes); err != nil {
		return fmt.Errorf("failed to create user indexes: %w", err)
	}

	// Records indexes
	recordIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "genre", Value: 1}}},
		{Keys: bson.D{{Key: "year", Value: 1}}},
		{Keys: bson.D{{Key: "format", Value: 1}}},
		{Keys: bson.D{{Key: "artist", Value: 1}}},
		{Keys: bson.D{{Key: "price", Value: 1}}},
		{Keys: bson.D{{Key: "archived", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	}
	if _, err := s.records.Indexes().CreateMany(ctx, recordIndexes); err != nil {
		return fmt.Errorf("failed to create record indexes: %w", err)
	}

	// Orders indexes
	orderIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	}
	if _, err := s.orders.Indexes().CreateMany(ctx, orderIndexes); err != nil {
		return fmt.Errorf("failed to create order indexes: %w", err)
	}

	return nil
}

// SeedData adds sample records to the store
func (s *MongoStore) SeedData() {
	seedRecords := []*models.Record{
		// Jazz (15 records)
		seedRecord("rec-001", "Kind of Blue", "Miles Davis", 1959, "Columbia", "CL 1355", models.FormatLP, models.GenreJazz, models.ConditionNM, 29.99, 5, "Essential modal jazz masterpiece"),
		seedRecord("rec-002", "A Love Supreme", "John Coltrane", 1965, "Impulse!", "A-77", models.FormatLP, models.GenreJazz, models.ConditionNM, 32.99, 4, "Spiritual jazz cornerstone"),
		seedRecord("rec-003", "Blue Train", "John Coltrane", 1958, "Blue Note", "BLP 1577", models.FormatLP, models.GenreJazz, models.ConditionMint, 44.99, 1, "Hard bop classic with all-star lineup"),
		seedRecord("rec-004", "Time Out", "The Dave Brubeck Quartet", 1959, "Columbia", "CL 1397", models.FormatLP, models.GenreJazz, models.ConditionNMMinus, 24.99, 6, "Features Take Five in 5/4 time"),
		seedRecord("rec-005", "The Shape of Jazz to Come", "Ornette Coleman", 1959, "Atlantic", "SD 1317", models.FormatLP, models.GenreJazz, models.ConditionVGPlus, 27.99, 3, "Free jazz pioneering album"),
		seedRecord("rec-006", "Mingus Ah Um", "Charles Mingus", 1959, "Columbia", "CL 1370", models.FormatLP, models.GenreJazz, models.ConditionNM, 26.99, 4, "Mingus masterpiece with gospel and blues roots"),
		seedRecord("rec-007", "Sunday at the Village Vanguard", "Bill Evans Trio", 1961, "Riverside", "RLP-376", models.FormatLP, models.GenreJazz, models.ConditionNMMinus, 34.99, 2, "Live piano trio perfection"),
		seedRecord("rec-008", "Saxophone Colossus", "Sonny Rollins", 1956, "Prestige", "PRLP 7079", models.FormatLP, models.GenreJazz, models.ConditionMint, 39.99, 3, "Includes St. Thomas and Blue 7"),
		seedRecord("rec-009", "Somethin' Else", "Cannonball Adderley", 1958, "Blue Note", "BLP 1595", models.FormatLP, models.GenreJazz, models.ConditionNM, 29.99, 5, "Miles Davis sits in on this session"),
		seedRecord("rec-010", "The Sidewinder", "Lee Morgan", 1964, "Blue Note", "BLP 4157", models.FormatLP, models.GenreJazz, models.ConditionVG, 19.99, 4, "Soul jazz hit with iconic title track"),
		seedRecord("rec-011", "Head Hunters", "Herbie Hancock", 1973, "Columbia", "KC 32731", models.FormatLP, models.GenreJazz, models.ConditionNMMinus, 22.99, 7, "Jazz-funk fusion breakthrough"),
		seedRecord("rec-012", "Getz/Gilberto", "Stan Getz & João Gilberto", 1964, "Verve", "V6-8545", models.FormatLP, models.GenreJazz, models.ConditionNM, 24.99, 5, "Bossa nova classic with Girl from Ipanema"),
		seedRecord("rec-013", "Monk's Music", "Thelonious Monk", 1957, "Riverside", "RLP 12-242", models.FormatLP, models.GenreJazz, models.ConditionVGPlus, 28.99, 3, "Features John Coltrane on tenor"),
		seedRecord("rec-014", "Moanin'", "Art Blakey & The Jazz Messengers", 1958, "Blue Note", "BLP 4003", models.FormatLP, models.GenreJazz, models.ConditionMint, 42.99, 2, "Hard bop essential with Lee Morgan"),
		seedRecord("rec-015", "Out to Lunch!", "Eric Dolphy", 1964, "Blue Note", "BLP 4163", models.FormatLP, models.GenreJazz, models.ConditionNM, 31.99, 3, "Avant-garde jazz milestone"),

		// Rock (20 records)
		seedRecord("rec-016", "The Dark Side of the Moon", "Pink Floyd", 1973, "Harvest", "SHVL 804", models.FormatLP, models.GenreRock, models.ConditionMint, 34.99, 3, "Psychedelic rock masterpiece"),
		seedRecord("rec-017", "Abbey Road", "The Beatles", 1969, "Apple", "PCS 7088", models.FormatLP, models.GenreRock, models.ConditionVGPlus, 39.99, 2, "The Beatles' final recorded album"),
		seedRecord("rec-018", "The Rise and Fall of Ziggy Stardust", "David Bowie", 1972, "RCA", "SF 8287", models.FormatLP, models.GenreRock, models.ConditionNM, 31.99, 5, "Glam rock concept album classic"),
		seedRecord("rec-019", "Purple Rain", "Prince", 1984, "Warner Bros.", "25110-1", models.FormatLP, models.GenreRock, models.ConditionNMMinus, 26.99, 6, "Prince's magnum opus"),
		seedRecord("rec-020", "Rumours", "Fleetwood Mac", 1977, "Warner Bros.", "BSK 3010", models.FormatLP, models.GenreRock, models.ConditionNM, 24.99, 8, "Breakup album that defined the 70s"),
		seedRecord("rec-021", "Led Zeppelin IV", "Led Zeppelin", 1971, "Atlantic", "SD 7208", models.FormatLP, models.GenreRock, models.ConditionMint, 36.99, 4, "Features Stairway to Heaven"),
		seedRecord("rec-022", "Nevermind", "Nirvana", 1991, "DGC", "DGCC-24425", models.FormatLP, models.GenreRock, models.ConditionNM, 29.99, 6, "Grunge breakthrough album"),
		seedRecord("rec-023", "OK Computer", "Radiohead", 1997, "Parlophone", "7243 8 55229 1 5", models.FormatLP, models.GenreRock, models.ConditionNMMinus, 27.99, 5, "Alternative rock art masterpiece"),
		seedRecord("rec-024", "The Doors", "The Doors", 1967, "Elektra", "EKS-74007", models.FormatLP, models.GenreRock, models.ConditionVG, 18.99, 3, "Debut with Light My Fire"),
		seedRecord("rec-025", "Born to Run", "Bruce Springsteen", 1975, "Columbia", "PC 33795", models.FormatLP, models.GenreRock, models.ConditionNM, 22.99, 4, "Springsteen's breakthrough"),
		seedRecord("rec-026", "London Calling", "The Clash", 1979, "CBS", "S CBS 8380", models.FormatLP, models.GenreRock, models.ConditionVGPlus, 26.99, 3, "Punk/post-punk double album"),
		seedRecord("rec-027", "Are You Experienced", "The Jimi Hendrix Experience", 1967, "Track", "612 001", models.FormatLP, models.GenreRock, models.ConditionNMMinus, 29.99, 4, "Psychedelic rock guitar revolution"),
		seedRecord("rec-028", "Pet Sounds", "The Beach Boys", 1966, "Capitol", "T 2458", models.FormatLP, models.GenreRock, models.ConditionNM, 34.99, 2, "Art pop production masterpiece"),
		seedRecord("rec-029", "Who's Next", "The Who", 1971, "Track", "2408 102", models.FormatLP, models.GenreRock, models.ConditionMint, 32.99, 3, "Features Baba O'Riley and Won't Get Fooled Again"),
		seedRecord("rec-030", "Paranoid", "Black Sabbath", 1970, "Vertigo", "VO 6", models.FormatLP, models.GenreRock, models.ConditionVGPlus, 24.99, 5, "Heavy metal foundation"),
		seedRecord("rec-031", "Joshua Tree", "U2", 1987, "Island", "U2-6", models.FormatLP, models.GenreRock, models.ConditionNM, 21.99, 6, "Stadium rock anthem collection"),
		seedRecord("rec-032", "Blood on the Tracks", "Bob Dylan", 1975, "Columbia", "PC 33235", models.FormatLP, models.GenreRock, models.ConditionNMMinus, 23.99, 4, "Dylan's emotional breakup album"),
		seedRecord("rec-033", "Appetite for Destruction", "Guns N' Roses", 1987, "Geffen", "GHS 24148", models.FormatLP, models.GenreRock, models.ConditionNM, 25.99, 7, "Hard rock rebirth in the late 80s"),
		seedRecord("rec-034", "Is This It", "The Strokes", 2001, "RCA", "07863 68003-1", models.FormatLP, models.GenreRock, models.ConditionMint, 27.99, 5, "Garage rock revival classic"),
		seedRecord("rec-035", "Fun House", "The Stooges", 1970, "Elektra", "EKS-74091", models.FormatLP, models.GenreRock, models.ConditionVG, 29.99, 2, "Raw proto-punk energy"),

		// Electronic (12 records)
		seedRecord("rec-036", "Discovery", "Daft Punk", 2001, "Virgin", "7243 8 10166 1 2", models.FormatLP, models.GenreElectronic, models.ConditionNMMinus, 24.99, 8, "French house at its finest"),
		seedRecord("rec-037", "Random Access Memories", "Daft Punk", 2013, "Columbia", "888837-16861", models.FormatLP, models.GenreElectronic, models.ConditionMint, 28.99, 10, "Grammy-winning disco-infused electronic"),
		seedRecord("rec-038", "Selected Ambient Works 85-92", "Aphex Twin", 1992, "Apollo", "AMB 3922", models.FormatLP, models.GenreElectronic, models.ConditionNM, 33.99, 4, "Ambient techno cornerstone"),
		seedRecord("rec-039", "Music Has the Right to Children", "Boards of Canada", 1998, "Warp", "WARPLP55", models.FormatLP, models.GenreElectronic, models.ConditionNM, 29.99, 3, "IDM classic with nostalgic samples"),
		seedRecord("rec-040", "Untrue", "Burial", 2007, "Hyperdub", "HDBCD002", models.FormatLP, models.GenreElectronic, models.ConditionNMMinus, 26.99, 5, "Dubstep masterpiece"),
		seedRecord("rec-041", "Homework", "Daft Punk", 1997, "Virgin", "7243 8 42609 1 0", models.FormatLP, models.GenreElectronic, models.ConditionVGPlus, 21.99, 4, "Daft Punk's debut with Around the World"),
		seedRecord("rec-042", "Play", "Moby", 1999, "Mute", "STUMM172", models.FormatLP, models.GenreElectronic, models.ConditionNM, 19.99, 6, "Licensing hit with blues samples"),
		seedRecord("rec-043", "Dummy", "Portishead", 1994, "Go! Discs", "828 522-1", models.FormatLP, models.GenreElectronic, models.ConditionNM, 27.99, 5, "Trip-hop Bristol sound"),
		seedRecord("rec-044", "Leftism", "Leftfield", 1995, "Hard Hands", "HANDLP2T", models.FormatLP, models.GenreElectronic, models.ConditionMint, 31.99, 3, "Progressive house classic"),
		seedRecord("rec-045", "Exit Planet Dust", "The Chemical Brothers", 1995, "Junior Boy's Own", "JBOLP5", models.FormatLP, models.GenreElectronic, models.ConditionVG, 22.99, 4, "Big beat pioneers debut"),
		seedRecord("rec-046", "Replica", "Oneohtrix Point Never", 2011, "Mexican Summer", "MEX 0603", models.FormatLP, models.GenreElectronic, models.ConditionNM, 24.99, 3, "Experimental vaporwave sample collage"),
		seedRecord("rec-047", "Rounds", "Four Tet", 2003, "Domino", "WIGLP100", models.FormatLP, models.GenreElectronic, models.ConditionNMMinus, 23.99, 5, "Folktronica glitchy beats"),

		// Hip-Hop (12 records)
		seedRecord("rec-048", "Enter the Wu-Tang (36 Chambers)", "Wu-Tang Clan", 1993, "Loud", "07863 66336-1", models.FormatLP, models.GenreHipHop, models.ConditionNMMinus, 29.99, 7, "Raw Staten Island hip-hop classic"),
		seedRecord("rec-049", "Illmatic", "Nas", 1994, "Columbia", "CK 57684", models.FormatLP, models.GenreHipHop, models.ConditionMint, 34.99, 2, "Queensbridge lyrical masterpiece"),
		seedRecord("rec-050", "Ready to Die", "The Notorious B.I.G.", 1994, "Bad Boy", "78612-73000-1", models.FormatLP, models.GenreHipHop, models.ConditionNM, 31.99, 4, "East Coast gangsta rap essential"),
		seedRecord("rec-051", "Midnight Marauders", "A Tribe Called Quest", 1993, "Jive", "01241-41481-1", models.FormatLP, models.GenreHipHop, models.ConditionNM, 27.99, 5, "Jazz rap perfection"),
		seedRecord("rec-052", "Madvillainy", "Madvillain", 2004, "Stones Throw", "STH 2065", models.FormatLP, models.GenreHipHop, models.ConditionNMMinus, 28.99, 3, "MF DOOM and Madlib collaboration"),
		seedRecord("rec-053", "The Low End Theory", "A Tribe Called Quest", 1991, "Jive", "01241-41459-1", models.FormatLP, models.GenreHipHop, models.ConditionVGPlus, 25.99, 4, "Bass-heavy jazz rap evolution"),
		seedRecord("rec-054", "Paid in Full", "Eric B. & Rakim", 1987, "4th & B'way", "BWAY 4004", models.FormatLP, models.GenreHipHop, models.ConditionVG, 19.99, 3, "Lyricism revolution"),
		seedRecord("rec-055", "Liquid Swords", "GZA", 1995, "Geffen", "GEF-24705", models.FormatLP, models.GenreHipHop, models.ConditionNM, 29.99, 4, "Chess-themed Wu-Tang solo classic"),
		seedRecord("rec-056", "It Takes a Nation of Millions", "Public Enemy", 1988, "Def Jam", "527 759-1", models.FormatLP, models.GenreHipHop, models.ConditionNM, 24.99, 5, "Political hip-hop bomb squad production"),
		seedRecord("rec-057", "My Beautiful Dark Twisted Fantasy", "Kanye West", 2010, "Def Jam", "B0014695-01", models.FormatLP, models.GenreHipHop, models.ConditionMint, 32.99, 6, "Maximalist rap opera"),
		seedRecord("rec-058", "Endtroducing.....", "DJ Shadow", 1996, "Mo' Wax", "MW 061", models.FormatLP, models.GenreHipHop, models.ConditionNM, 26.99, 4, "Entirely sample-based instrumental hip-hop"),
		seedRecord("rec-059", "Aquemini", "OutKast", 1998, "LaFace", "73008-26059-1", models.FormatLP, models.GenreHipHop, models.ConditionNMMinus, 27.99, 5, "Southern hip-hop cinematic masterpiece"),

		// Soul (8 records)
		seedRecord("rec-060", "What's Going On", "Marvin Gaye", 1971, "Tamla", "TS 310", models.FormatLP, models.GenreSoul, models.ConditionVGPlus, 22.99, 3, "Motown social consciousness masterpiece"),
		seedRecord("rec-061", "I Never Loved a Man the Way I Love You", "Aretha Franklin", 1967, "Atlantic", "SD 8139", models.FormatLP, models.GenreSoul, models.ConditionNM, 24.99, 4, "Queen of Soul with Respect"),
		seedRecord("rec-062", "Hot Buttered Soul", "Isaac Hayes", 1969, "Enterprise", "ENS-4005", models.FormatLP, models.GenreSoul, models.ConditionMint, 28.99, 2, "Epic cinematic soul"),
		seedRecord("rec-063", "Let's Get It On", "Marvin Gaye", 1973, "Tamla", "TS 326", models.FormatLP, models.GenreSoul, models.ConditionNMMinus, 21.99, 5, "Sensual soul classic"),
		seedRecord("rec-064", "In the Wee Small Hours", "Frank Sinatra", 1955, "Capitol", "W 581", models.FormatLP, models.GenreSoul, models.ConditionVG, 16.99, 3, "Concept album about loneliness"),
		seedRecord("rec-065", "Back to Black", "Amy Winehouse", 2006, "Island", "171 163-8", models.FormatLP, models.GenreSoul, models.ConditionNM, 25.99, 6, "Modern soul revival with Mark Ronson"),
		seedRecord("rec-066", "Voodoo", "D'Angelo", 2000, "Virgin", "7243 8 48747 1 0", models.FormatLP, models.GenreSoul, models.ConditionNM, 29.99, 4, "Neo-soul experimental masterpiece"),
		seedRecord("rec-067", "Shaft", "Isaac Hayes", 1971, "Enterprise", "ENS-2-5002", models.FormatLP, models.GenreSoul, models.ConditionVGPlus, 23.99, 3, "Blaxploitation soundtrack double LP"),

		// Funk (10 records)
		seedRecord("rec-068", "Maggot Brain", "Funkadelic", 1971, "Westbound", "WB 2007", models.FormatLP, models.GenreFunk, models.ConditionNM, 35.99, 2, "Ten-minute title track guitar opus"),
		seedRecord("rec-069", "Mothership Connection", "Parliament", 1975, "Casablanca", "NBLP 7022", models.FormatLP, models.GenreFunk, models.ConditionNMMinus, 26.99, 4, "P-Funk space opera"),
		seedRecord("rec-070", "There's a Riot Goin' On", "Sly & The Family Stone", 1971, "Epic", "KE 30986", models.FormatLP, models.GenreFunk, models.ConditionVG, 21.99, 3, "Dark psychedelic funk"),
		seedRecord("rec-071", "Super Fly", "Curtis Mayfield", 1972, "Curtom", "CRS 8014", models.FormatLP, models.GenreFunk, models.ConditionNM, 24.99, 5, "Blaxploitation soundtrack classic"),
		seedRecord("rec-072", "Sex Machine", "James Brown", 1970, "King", "KS 1111", models.FormatLP, models.GenreFunk, models.ConditionVGPlus, 19.99, 4, "Live funk foundation"),
		seedRecord("rec-073", "Fire", "The Ohio Players", 1974, "Westbound", "WB 5003", models.FormatLP, models.GenreFunk, models.ConditionMint, 22.99, 3, "Funk with killer guitar riffs"),
		seedRecord("rec-074", "Standing on the Verge of Getting It On", "Funkadelic", 1974, "Westbound", "WB 2015", models.FormatLP, models.GenreFunk, models.ConditionNM, 27.99, 4, "Eddie Hazel guitar showcase"),
		seedRecord("rec-075", "Fresh", "Sly & The Family Stone", 1973, "Epic", "KE 32453", models.FormatLP, models.GenreFunk, models.ConditionNMMinus, 23.99, 5, "Post-Riot stripped down funk"),
		seedRecord("rec-076", "The Payback", "James Brown", 1973, "Polydor", "PD-2-3007", models.FormatLP, models.GenreFunk, models.ConditionVG, 18.99, 3, "Hardest working man in show business"),
		seedRecord("rec-077", "Afwk", "Bootsy's Rubber Band", 1976, "Warner Bros.", "BS 2963", models.FormatLP, models.GenreFunk, models.ConditionNM, 25.99, 4, "Bootsy Collins space bass"),

		// Blues (8 records)
		seedRecord("rec-078", "At Last!", "Etta James", 1960, "Argo", "LP 4003", models.FormatLP, models.GenreBlues, models.ConditionVGPlus, 19.99, 2, "Includes timeless At Last"),
		seedRecord("rec-079", "King of the Delta Blues Singers", "Robert Johnson", 1961, "Columbia", "CL 1654", models.FormatLP, models.GenreBlues, models.ConditionNM, 32.99, 3, "Essential pre-war blues collection"),
		seedRecord("rec-080", "Born Under a Bad Sign", "Albert King", 1967, "Stax", "STS 7235", models.FormatLP, models.GenreBlues, models.ConditionNMMinus, 24.99, 4, "Electric blues cornerstone"),
		seedRecord("rec-081", "Live at the Regal", "B.B. King", 1965, "ABC", "ABC 509", models.FormatLP, models.GenreBlues, models.ConditionMint, 29.99, 2, "The thrill is live"),
		seedRecord("rec-082", "Texas Flood", "Stevie Ray Vaughan", 1983, "Epic", "E 38748", models.FormatLP, models.GenreBlues, models.ConditionNM, 26.99, 5, "Modern blues guitar explosion"),
		seedRecord("rec-083", "Lady Sings the Blues", "Billie Holiday", 1956, "Clef", "MGC 169", models.FormatLP, models.GenreBlues, models.ConditionVG, 15.99, 3, "Verve years compilation"),
		seedRecord("rec-084", "Hard Again", "Muddy Waters", 1977, "Blue Sky", "JZ 34939", models.FormatLP, models.GenreBlues, models.ConditionNM, 22.99, 4, "Johnny Winter produced comeback"),
		seedRecord("rec-085", "The Healer", "John Lee Hooker", 1989, "Chameleon", "D2-74808", models.FormatLP, models.GenreBlues, models.ConditionNMMinus, 20.99, 5, "Late career Grammy-winning collaboration"),

		// Classical (15 records)
		seedRecord("rec-086", "The Four Seasons", "Antonio Vivaldi", 1959, "Decca", "SXL 2001", models.FormatLP, models.GenreClassical, models.ConditionNMMinus, 18.99, 3, "Baroque violin concertos"),
		seedRecord("rec-087", "Symphony No. 9 'Choral'", "Ludwig van Beethoven", 1963, "Deutsche Grammophon", "SLPM 138801", models.FormatLP, models.GenreClassical, models.ConditionNM, 21.99, 4, "Karajan conducts the Berlin Philharmonic"),
		seedRecord("rec-088", "The Well-Tempered Clavier", "Johann Sebastian Bach", 1965, "Deutsche Grammophon", "2720 006", models.FormatLP, models.GenreClassical, models.ConditionMint, 24.99, 2, "Glenn Gould piano interpretation"),
		seedRecord("rec-089", "Requiem", "Wolfgang Amadeus Mozart", 1975, "Philips", "6769 011", models.FormatLP, models.GenreClassical, models.ConditionNM, 19.99, 3, "Böhm conducting Vienna Philharmonic"),
		seedRecord("rec-090", "The Planets", "Gustav Holst", 1971, "EMI", "SLS 986", models.FormatLP, models.GenreClassical, models.ConditionVGPlus, 16.99, 4, "Boult conducts British classic"),
		seedRecord("rec-091", "Symphony No. 5", "Gustav Mahler", 1973, "Deutsche Grammophon", "2709 077", models.FormatLP, models.GenreClassical, models.ConditionNMMinus, 22.99, 3, "Bernstein's emotional interpretation"),
		seedRecord("rec-092", "Cello Suites", "Johann Sebastian Bach", 1961, "EMI", "ALP 1933", models.FormatLP, models.GenreClassical, models.ConditionNM, 26.99, 2, "Pablo Casals definitive recording"),
		seedRecord("rec-093", "La Mer / Nocturnes", "Claude Debussy", 1964, "Decca", "SXL 6113", models.FormatLP, models.GenreClassical, models.ConditionVG, 14.99, 5, "Impressionist orchestral works"),
		seedRecord("rec-094", "The Rite of Spring", "Igor Stravinsky", 1958, "Columbia", "ML 5192", models.FormatLP, models.GenreClassical, models.ConditionNM, 20.99, 4, "Stravinsky conducts his revolutionary ballet"),
		seedRecord("rec-095", "Piano Concerto No. 2", "Sergei Rachmaninoff", 1972, "Decca", "SXL 6518", models.FormatLP, models.GenreClassical, models.ConditionMint, 23.99, 3, "Ashkenazy with Concertgebouw"),
		seedRecord("rec-096", "Symphony No. 7", "Anton Bruckner", 1966, "Deutsche Grammophon", "139122", models.FormatLP, models.GenreClassical, models.ConditionNMMinus, 18.99, 4, "Karajan and Berlin Philharmonic"),
		seedRecord("rec-097", "Tristan und Isolde", "Richard Wagner", 1972, "Deutsche Grammophon", "2720 021", models.FormatLP, models.GenreClassical, models.ConditionNM, 27.99, 2, "Böhm conducting Wagner's masterpiece"),
		seedRecord("rec-098", "Adagio for Strings", "Samuel Barber", 1982, "EMI", "EL 27 0344 1", models.FormatLP, models.GenreClassical, models.ConditionNM, 15.99, 6, "Schwarz conducts emotional modern classic"),
		seedRecord("rec-099", "Symphonie Fantastique", "Hector Berlioz", 1969, "Philips", "6500 094", models.FormatLP, models.GenreClassical, models.ConditionVGPlus, 17.99, 3, "Davis conducts revolutionary program symphony"),
		seedRecord("rec-100", "Goldberg Variations", "Johann Sebastian Bach", 1955, "Columbia", "ML 5060", models.FormatLP, models.GenreClassical, models.ConditionNM, 29.99, 2, "Glenn Gould's legendary debut recording"),
	}

	for _, r := range seedRecords {
		_ = s.SaveRecord(context.Background(), r)
	}
}

func seedRecord(id, title, artist string, year int, label, catalogNumber string, format models.Format, genre models.Genre, condition models.Condition, price float64, stock int, description string) *models.Record {
	r := models.NewRecord(models.RecordParams{
		Title:         title,
		Artist:        artist,
		Year:          year,
		Label:         label,
		CatalogNumber: catalogNumber,
		Format:        format,
		Genre:         genre,
		Condition:     condition,
		Price:         price,
		Stock:         stock,
		Description:   description,
	})
	r.ID = id
	return r
}

// UserStore methods

func (s *MongoStore) GetUser(ctx context.Context, id string) (*models.User, bool, error) {
	var user models.User
	err := s.users.FindOne(ctx, bson.M{"id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &user, true, nil
}

func (s *MongoStore) SaveUser(ctx context.Context, u *models.User) error {
	opts := options.Replace().SetUpsert(true)
	_, err := s.users.ReplaceOne(ctx, bson.M{"id": u.ID}, u, opts)
	return err
}

func (s *MongoStore) GetUserByEmail(ctx context.Context, email string) (*models.User, bool, error) {
	var user models.User
	err := s.users.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &user, true, nil
}

func (s *MongoStore) GetAllUsers(ctx context.Context, f UserFilter) (users []*models.User, total int, err error) {
	count, err := s.users.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	params := pagination.Params{Page: f.BaseFilter.Page, PerPage: f.BaseFilter.PerPage}.Defaults()
	limit, skip := params.LimitSkip()

	cursor, err := s.users.Find(ctx, bson.M{}, options.Find().SetLimit(int64(limit)).SetSkip(int64(skip)))
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if cerr := cursor.Close(ctx); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}
	return users, int(count), nil
}

// RecordStore methods

func (s *MongoStore) GetRecord(ctx context.Context, id string) (*models.Record, bool, error) {
	var record models.Record
	err := s.records.FindOne(ctx, bson.M{"id": id}).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &record, true, nil
}

func (s *MongoStore) SaveRecord(ctx context.Context, r *models.Record) error {
	opts := options.Replace().SetUpsert(true)
	_, err := s.records.ReplaceOne(ctx, bson.M{"id": r.ID}, r, opts)
	return err
}

// DecrementStock atomically decrements the stock of a record by the given quantity.
// Returns true if the stock was successfully decremented, false if insufficient stock.
func (s *MongoStore) DecrementStock(ctx context.Context, id string, quantity int) (bool, error) {
	filter := bson.M{
		"id":    id,
		"stock": bson.M{"$gte": quantity},
	}
	update := bson.M{
		"$inc": bson.M{"stock": -quantity},
		"$set": bson.M{"updated_at": time.Now()},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var record models.Record
	err := s.records.FindOneAndUpdate(ctx, filter, update, opts).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *MongoStore) GetAllRecords(ctx context.Context) (records []*models.Record, err error) {
	cursor, err := s.records.Find(ctx, bson.M{"archived": false})
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := cursor.Close(ctx); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *MongoStore) GetFilteredRecords(ctx context.Context, f RecordFilter) (records []*models.Record, total int, err error) {
	filter := bson.M{"archived": false}
	if len(f.Genres) > 0 {
		filter["genre"] = bson.M{"$in": f.Genres}
	}
	if f.Decade != 0 {
		filter["year"] = bson.M{
			"$gte": f.Decade,
			"$lt":  f.Decade + 10,
		}
	}
	if len(f.Formats) > 0 {
		filter["format"] = bson.M{"$in": f.Formats}
	}
	if len(f.Conditions) > 0 {
		filter["condition"] = bson.M{"$in": f.Conditions}
	}
	if f.Artist != "" {
		filter["artist"] = f.Artist
	}
	if f.MinPrice > 0 || f.MaxPrice > 0 {
		priceFilter := bson.M{}
		if f.MinPrice > 0 {
			priceFilter["$gte"] = f.MinPrice
		}
		if f.MaxPrice > 0 {
			priceFilter["$lte"] = f.MaxPrice
		}
		filter["price"] = priceFilter
	}
	if f.InStock {
		filter["stock"] = bson.M{"$gt": 0}
	}

	count, err := s.records.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find()

	sort := bson.D{}
	switch f.Sort {
	case "price_asc":
		sort = append(sort, bson.E{Key: "price", Value: 1})
	case "price_desc":
		sort = append(sort, bson.E{Key: "price", Value: -1})
	case "year_asc":
		sort = append(sort, bson.E{Key: "year", Value: 1})
	case "year_desc":
		sort = append(sort, bson.E{Key: "year", Value: -1})
	case "created_desc":
		sort = append(sort, bson.E{Key: "created_at", Value: -1})
	}
	if len(sort) > 0 {
		opts.SetSort(sort)
	}

	params := pagination.Params{Page: f.BaseFilter.Page, PerPage: f.BaseFilter.PerPage}.Defaults()
	limit, skip := params.LimitSkip()

	opts.SetLimit(int64(limit))
	opts.SetSkip(int64(skip))

	cursor, err := s.records.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if cerr := cursor.Close(ctx); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err = cursor.All(ctx, &records); err != nil {
		return nil, 0, err
	}
	return records, int(count), nil
}

// OrderStore methods

func (s *MongoStore) GetOrder(ctx context.Context, id string) (*models.Order, bool, error) {
	var order models.Order
	err := s.orders.FindOne(ctx, bson.M{"id": id}).Decode(&order)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &order, true, nil
}

func (s *MongoStore) SaveOrder(ctx context.Context, o *models.Order) error {
	opts := options.Replace().SetUpsert(true)
	_, err := s.orders.ReplaceOne(ctx, bson.M{"id": o.ID}, o, opts)
	return err
}

func (s *MongoStore) GetOrdersByUser(ctx context.Context, f OrderFilter) (orders []*models.Order, total int, err error) {
	filter := bson.M{"user_id": f.UserID}
	if f.Status != "" {
		filter["status"] = f.Status
	}

	count, err := s.orders.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	params := pagination.Params{Page: f.BaseFilter.Page, PerPage: f.BaseFilter.PerPage}.Defaults()
	limit, skip := params.LimitSkip()

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(skip))

	cursor, err := s.orders.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if cerr := cursor.Close(ctx); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err = cursor.All(ctx, &orders); err != nil {
		return nil, 0, err
	}
	return orders, int(count), nil
}

func (s *MongoStore) GetAllOrders(ctx context.Context, f AllOrdersFilter) (orders []*models.Order, total int, err error) {
	count, err := s.orders.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	params := pagination.Params{Page: f.BaseFilter.Page, PerPage: f.BaseFilter.PerPage}.Defaults()
	limit, skip := params.LimitSkip()

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(skip))

	cursor, err := s.orders.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if cerr := cursor.Close(context.Background()); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if err = cursor.All(ctx, &orders); err != nil {
		return nil, 0, err
	}
	return orders, int(count), nil
}

// Close disconnects from MongoDB
func (s *MongoStore) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}
