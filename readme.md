### TODOs
- Test with bigger file (like lecture video)
    - Did not work! Google has way too small limit on max file size. 40 Min lecture video was too big
    - Possible attempt: split big wav into many small wavs using ffmpeg, make many parallel small requests instead of one big one
    - Fail! Restrictions are so terrible, that it might be better to just use the non-longrunningrecognize requests instead
        - running takes too long, would be a timeout in rl

- Upload of all the small files uses the most time:
    - check if its faster to upload one big file