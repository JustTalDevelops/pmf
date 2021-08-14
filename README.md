# pmf
A implementation for the legacy PocketMine chunk format, and built in methods to convert it to the latest mcworld format.

I really just speedran this in a day so I could convert the old Origins Survival Games map to the latest world format for use in Dragonfly, 
but someone else might find this useful.

# Block entity conversion
This one was a bit tricky, because of the way block entities, also known as tiles,
are stored in PMF. There's a tiles.yml file that contains tile data, however the formatting
is different to modern tile data, so we must implement tile support one by one.

Currently, I've implemented support for:

- Signs

# Legacy PM image
![](./images/old_image.png)

# Updated Bedrock image
![](./images/new_image.png)