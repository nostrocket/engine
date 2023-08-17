package merits

//when creating a merit request, the user can request BTC or merits.
//if Merits, they are created at the current price (starts at 1sat per merit)
//if they want BTC, the merit request is offered for sale (after approval).
//The process of selling:
//the number of merits issued to the merit request starts at 20% of the current value
//so for a merit price of 1 sat, a merit request for 10000 sats would be converted to 2000 merits.
//this 2000 goes up by 10% every block until sold.
//so the user gets paid the amount of their request in sats, but the number of merits recieved by the buyer is subject
//to a dutch auction.

//Once a sale has been made, this is the new global price of merits for future merit requests and sales.
