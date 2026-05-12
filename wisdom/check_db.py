import sqlite3
import sys

def check_db():
    conn = sqlite3.connect('C:/Users/jesus/wisdom/wisdom/wisdom.db')
    cursor = conn.cursor()
    cursor.execute("SELECT name FROM sqlite_master WHERE type='table';")
    tables = cursor.fetchall()
    print("Tables:", tables)
    
    if len(tables) > 0:
        cursor.execute("SELECT * FROM namespaces;")
        print("Namespaces:", cursor.fetchall())
    
    conn.close()

if __name__ == '__main__':
    check_db()
