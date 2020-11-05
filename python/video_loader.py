import requests
import json
import subprocess
import os

TEST_LINK = 'https://d1hfnhxf88o06h.cloudfront.net/content/aa214783-ffbe-5b1f-82f4-0ed2ee89b483/20/11/02/03/aa214783-ffbe-5b1f-82f4-0ed2ee89b483_1_201102T032528572Z.mp4?Expires=1604330728&Signature=RhZtUX~PAabpCWMKlIuV85bhW4Yu7g8ZnwBR1Qr4pKWFd8T934wlJpplHuJyBMhm0D8dT0c6V6aBujSEsp3PL5~G7k-BTNlrU84UVVi9Ei9jPa78O5HkrfBwV4IMPc1syoMh8G0sxdBL3qwlZhJQF~QZggcCM~BdYvZC~vAPVJwm9EiaxEw0e~Pu5oD~DOuFyWa~82-xQJKJo7bt329DFCSVD4ikELY~ilJTyRw8pPY~8Jj05-DxwsX918UEjIfFi3Mp8TkH0fYGtBoQ-kf7K-2dtQs0CqdwJJZYiFBDBu6FgBj3L5leRTdSn9YjuSTTeH73o8M22sVRDdWbCQ1OVA__&Key-Pair-Id=APKAIOBDBIMXUOQOBYVA'

def load_video(video_url):
    downloaded_video = requests.get(video_url)
    video_name = './files/video.mp4'
    with open(video_name, 'wb') as videofile:
        videofile.write(downloaded_video.content)
    return video_name

def split_audio(video_name):
    command = f'ffmpeg -i {video_name} -ab 160k -ac 2 -ar 44100 -vn audio.wav'

    subprocess.call(command, shell=True)

def delete_files_in_folder(folder):
    for the_file in os.listdir(folder):
        file_path = os.path.join(folder, the_file)
        try:
            if os.path.isfile(file_path):
                os.unlink(file_path)
        except Exception as e:
            print(e)

def load_and_extract_audio_delete_files(video_url):
    vid = load_video(video_url)
    split_audio(vid)
    delete_files_in_folder('./files')
    


if __name__=='__main__':
    load_and_extract_audio_delete_files(TEST_LINK)