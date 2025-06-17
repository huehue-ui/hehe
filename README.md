# Setup service:
1. Clone the repository:
```bash
git clone git@github.com:TechLead-War/campaignservice.git
```
2. Run migrations.
```bash
migrate -path migrations -database "postgres://postgres:password@localhost:5432/campaign_service?sslmode=disable" up
```

3. Seed some data, with the following command:
```bash

go run seed.go -records=2000 -workers=20
```

4. Run the service:
```bash

air
```

### Note:<br>
This repo is tested with Go 1.23.5 and Postgres 15.4. With a 30L+ records.