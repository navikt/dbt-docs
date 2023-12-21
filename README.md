# NAV DBT dokumentasjon

Felles katalogløsning for generert DBT docs.

- [Prod](https://dbt.intern.nav.no)
- [Dev](https://dbt.intern.dev.nav.no)

## Publisering

Publisering gjøres med en HTTP `PUT` til `https://{HOST}/docs/{TEAM}/{DBT_PROSJEKT}`.

> [!WARNING]
> HTTP `PUT` operasjonen vil erstatte alt du tidligere har publisert for DBT prosjektet du prøver å oppdatere.

Man kan bruke HTTP `PATCH` mot samme endepunkt for å gjøre endringer på enkeltfiler.

- `{HOST}` erstattes med miljøet du vil publisere til
    - For prod: `dbt.intern.nav.no`
    - For dev: `dbt.intern.dev.nav.no`
- `{TEAM}` erstattes med navnet på teamet som eier dbt-prosjektet
- `{DBT_PROSJEKT}` erstattes med navnet på dbt prosjektet

Alle følgende genererte filer for dokumentasjonen er nødvendig:

- `index.html`
- `catalog.json`
- `manifest.json`

Under er eksempler på publisering med [Curl](#eksempel-med-curl) og [Python](#eksempel-med-python).
Begge eksemplene forutsetter at kommando eller skript kjøres fra katalogen med filene, og igjen må `{HOST}`, `{TEAM}` og `{DBT_PROSJEKT}` erstattes.

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

## Utvikling

### Generer CSS

Du kan manuelt generere CSS med kommandoen nedenfor.
Kommandoen blir også kjørt automatisk ved bruk av `Air`.

    npx tailwindcss --postcss -i assets/css/input.css -o assets/css/main.css

### Kjøre lokalt med Air

    air
