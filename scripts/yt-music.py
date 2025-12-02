#!/usr/bin/env python3

from ytmusicapi import YTMusic
import difflib

yt_music = YTMusic()


def find_yt_music_url(track_name, album_name, artist_name):
    search_query = f'{track_name} {album_name} {artist_name}'
    res = yt_music.search(search_query, filter='songs')

    best_match_index = -1
    best_match_ratio = 0.0
    for index, song in enumerate(res):
        # print(song['title'], '-', song['album']['name'], '-', song['artists'][0]['name'])
        title = song['title']
        album = song['album']['name']
        artists = [artist['name'] for artist in song['artists']]
        artist_ratios = [difflib.SequenceMatcher(None, artist_name.lower(), artist.lower()).ratio() for artist in artists]
        similarity_ratio = difflib.SequenceMatcher(None, track_name.lower(), title.lower()).ratio()
        similarity_ratio += difflib.SequenceMatcher(None, album_name.lower(), album.lower()).ratio() + max(artist_ratios)
        if similarity_ratio > best_match_ratio:
            best_match_ratio = similarity_ratio
            best_match_index = index

    if best_match_index == -1:
        best_match_index = 0

    return f'https://music.youtube.com/watch?v={res[best_match_index]["videoId"]}'


if __name__ == '__main__':
    import sys
    if len(sys.argv) != 4:
        print('Usage: yt-music.py <track name> <album name> <artist name>')
        sys.exit(1)
    track_name, album_name, artist_name = sys.argv[1:]
    print(find_yt_music_url(track_name, album_name, artist_name), end='')
