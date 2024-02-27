# soundcloud roulette

Randomly creates shortened soundcloud links (format: `https://on.soundcloud.com/h5AtN`) and loads them as an embed if they resolve.

![screenshot showing a button and an embedded soundcloud player on a dark background](https://github.com/lyramakesmusic/soundcloud-roulette/blob/main/soundcloud%20roulette.png)

## Install and run:

You will need to install Go, and download these files into a folder. `cd` into the folder and run `go run .`, then open http://localhost:8080/ in your browser.

The included .exe file is compiled directly from source and safe to run if you don't have Go installed, but it is **always safer** to compile it yourself!! Running random .exes is usually not a very good idea.

## Stats for Nerds

out of 10000 random urls, 2325 of them resolved to a playable song, 641 of them resolved to a "track removed" page.

extrapolating a little bit- that's ~23% of all 916m possible shortened links resolving to a song, or ~210m songs. ~21% of all tracks have been deleted.
