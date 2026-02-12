import sys
import os

# ANSI color codes for X and O
RED = '\033[91m'
BLUE = '\033[94m'
ENDC = '\033[0m'


def clear_screen():
    os.system('cls' if os.name == 'nt' else 'clear')


def print_board(board):
    print("\n    1   2   3")
    print("  +---+---+---+")
    for idx, row in enumerate(board):
        row_display = []
        for cell in row:
            if cell == 'X':
                row_display.append(f"{RED}X{ENDC}")
            elif cell == 'O':
                row_display.append(f"{BLUE}O{ENDC}")
            else:
                row_display.append(' ')
        print(f"{idx+1} | " + " | ".join(row_display) + " |")
        print("  +---+---+---+")


def check_winner(board, player):
    for row in board:
        if all([cell == player for cell in row]):
            return True
    for col in range(3):
        if all([board[row][col] == player for row in range(3)]):
            return True
    if all([board[i][i] == player for i in range(3)]):
        return True
    if all([board[i][2 - i] == player for i in range(3)]):
        return True
    return False


def is_full(board):
    return all([cell != ' ' for row in board for cell in row])


def main():
    board = [[' ' for _ in range(3)] for _ in range(3)]
    current_player = 'X'
    clear_screen()
    print("""
Welcome to Tic Tac Toe!

How to play:
- Players take turns entering the row and column number (e.g. 1 2) to place their mark.
- The first player to get 3 in a row (horizontally, vertically, or diagonally) wins.
- Enter numbers between 1 and 3 for both row and column.
""")
    input("Press Enter to start...")
    while True:
        clear_screen()
        print_board(board)
        print(f"Player {RED if current_player == 'X' else BLUE}{current_player}{ENDC}'s turn.")
        try:
            move = input("Enter row and column (e.g. 1 2): ")
            row, col = map(int, move.strip().split())
            if row < 1 or row > 3 or col < 1 or col > 3:
                print("Invalid input. Please enter numbers between 1 and 3.")
                input("Press Enter to continue...")
                continue
            if board[row-1][col-1] != ' ':
                print("Cell already taken. Try again.")
                input("Press Enter to continue...")
                continue
            board[row-1][col-1] = current_player
        except Exception:
            print("Invalid input format. Please enter two numbers separated by a space.")
            input("Press Enter to continue...")
            continue
        if check_winner(board, current_player):
            clear_screen()
            print_board(board)
            print(f"Player {RED if current_player == 'X' else BLUE}{current_player}{ENDC} wins! Congratulations!")
            break
        if is_full(board):
            clear_screen()
            print_board(board)
            print("It's a draw!")
            break
        current_player = 'O' if current_player == 'X' else 'X'

if __name__ == "__main__":
    main()
