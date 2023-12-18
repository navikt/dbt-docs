# NAV DBT dokumentasjon
Felles katalogløsning for generert dbt docs.

- [Prod](https://dbt.intern.nav.no)
- [Dev](https://dbt.intern.dev.nav.no)

## Publisering
Publisering gjøres med en http `PUT` til https://{HOST}/docs/{TEAM}/{DBT_PROSJEKT}. 
Merk: Denne operasjonen vil erstatte alt du tidligere har publisert for dbt prosjektet du prøver å oppdatere.
Hvis du i stedet ønsker å legge til filer gjøres det en http `PATCH` mot samme api.

- `{HOST}` erstattes med miljøet du vil publisere til
    - For prod: `dbt.intern.nav.no`
    - For dev: `dbt.intern.dev.nav.no`
- `{TEAM}` erstattes med navnet på teamet som eier dbt-prosjektet
- `{DBT_PROSJEKT}` erstattes med navnet på dbt prosjektet

Alle følgende genererte filer for dokumentasjonen er nødvendig:

- `index.html`
- `catalog.json`
- `manifest.json`

Under er eksempler på publisering med [curl](#eksempel-med-curl) og [python](#eksempel-med-python).
Begge eksemplene forutsetter at kommando eller skript kjøres fra mappen med filene listet opp over, og igjen må `{HOST}`, `{TEAM}` og `{DBT_PROSJEKT}` erstattes.

### Eksempel med curl
```sh
curl -X PUT \
    -F manifest.json=@manifest.json \
    -F catalog.json=@catalog.json \ 
    -F index.html=@index.html \
    https://{HOST}/docs/{TEAM}/{DBT_PROSJEKT}
```

### Eksempel med python
```python
import os
import requests

files = [
    "manifest.json",
    "catalog.json",
    "index.html",
]

multipart_form_data={}
for file_path in files:
    file_name = os.path.basename(file_path)
    with open(file_path, "rb") as file:
        file_contents = file.read()
        multipart_form_data[file_path] = (file_name, file_contents)

res = requests.put("https://{HOST}/docs/{TEAM}/{DBT_PROSJEKT}", files=multipart_form_data)
res.raise_for_status()
```
