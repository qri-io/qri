# Set a body containing a config field and a secret
ds = dataset.latest()
prev_body = ds.body
ds.body = [['Name', config.get('animal_name')],
           ['Sound', secrets.get('animal_sound')]]
dataset.commit(ds)
