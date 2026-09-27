package mtgmatcher

// Game names a game by the name its datastore loader registers under. It is
// the one spelling go-mtgban has for a game: flags, workflow names, bucket
// paths and the game every dump publishes all carry it as it is.
type Game string

// The games go-mtgban prices. Naming one here does not make it loadable: a
// game still exists only once something imports its package.
const (
	GameMagic         Game = "magic"
	GameLorcana       Game = "lorcana"
	GameRiftbound     Game = "riftbound"
	GameOnePiece      Game = "onepiece"
	GameYuGiOh        Game = "yugioh"
	GameFleshAndBlood Game = "fleshandblood"
	GamePokemon       Game = "pokemon"
	GameGundam        Game = "gundam"
	GamePalworld      Game = "palworld"
)

// AllGames is every game above, in a settled order: the official list, for a
// caller that has to run through all of them rather than name one. A game
// added to the constants belongs here too.
var AllGames = []Game{
	GameMagic,
	GameLorcana,
	GameRiftbound,
	GameOnePiece,
	GameYuGiOh,
	GameFleshAndBlood,
	GamePokemon,
	GameGundam,
	GamePalworld,
}
