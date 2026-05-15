/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package auth

const (
	clientID = "00000000402b5328"
	scope    = "service::user.auth.xboxlive.com::MBI_SSL"

	deviceCodeURL = "https://login.live.com/oauth20_connect.srf"
	tokenURL      = "https://login.live.com/oauth20_token.srf"
	xblURL        = "https://user.auth.xboxlive.com/user/authenticate"
	xstsURL       = "https://xsts.auth.xboxlive.com/xsts/authorize"
	mojangLogin   = "https://api.minecraftservices.com/authentication/login_with_xbox"
	mcProfileURL  = "https://api.minecraftservices.com/minecraft/profile"
)
