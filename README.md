## Fridge/Food Discord Bot

The main goal of this project is to gain comfort using both Golang and the Gemini API

### What does this bot do?

A basic Discord bot that scans images of food (e.g groceries). Then, using the Gemini API, recognizes the food, estimates the nutrition facts as well as the likely expiration date. This information is stored in a database and eventually the bot should ping the user when the food is expected to spoil.

### Current commands
!ask <img>: Queries the Gemini API without saving the food to the SQLite db
!scan <img>: Queries the Gemini API and saves the result to db
!get: Gets all food information currently stored for that user in the db
!clearall: Removes all food information currently stored for that user in the db

### Future commands
!clear [food name in natural language]: Uses the gemini api to select which foods to delete
!makefood [natural language prompt]: Comes up with a recipe using stored foods


- React to message to tell the bot how the food will be stored to better schedule ping