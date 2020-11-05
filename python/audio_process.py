from scipy.io import wavfile
import numpy as np
import matplotlib.pyplot as plt


def process_file(wave_file):
    samplerate, data = wavfile.read(wave_file)
    # print(f'data: \n{data}')
    # print(f'data.shape: {data.shape}')

    # print(f'\n\nsamplerate: {samplerate}')
    print(f'Length; {sound_length(samplerate, data)}')
    plot_wav(samplerate, data)

def sound_length(samplerate, data):
    return data.shape[0]/samplerate

def plot_wav(samplerate, data):
    time = np.linspace(0., sound_length(samplerate, data), data.shape[0])
    plt.plot(time, data[:, 0], label="Left channel")
    plt.plot(time, data[:, 1], label="Right channel")
    plt.legend()
    plt.xlabel("Time [s]")
    plt.ylabel("Amplitude")
    plt.show()
    

if __name__=='__main__':
    process_file('./audio.wav')
    