package cardmarket

// lorcanaRenames gives the name the datastore knows a Cardmarket Lorcana
// product by, where Cardmarket sells the printing under another title.
var lorcanaRenames = map[int]string{
	804164: "Moana - Adventurer of Land and Sea", // "Vaiana", Moana's European title
}

// lorcanaNonEnglish holds the products Cardmarket shelves among the English
// cards that were printed in another language alone: the Chinese Exclusive
// Snow White and Stitch promos.
var lorcanaNonEnglish = map[int]bool{
	835454: true, 895480: true,
}
