package tcgplayer

import (
	"github.com/kodabb/go-mtgban/mtgban"
)

type responseChan struct {
	cardId string
	entry  mtgban.InventoryEntry
	bl     mtgban.BuylistEntry
}
