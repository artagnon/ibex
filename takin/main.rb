require 'net/http'
require 'json'
require 'ncursesw'

module Ncurses
  def rows
    lame, lamer = [], []
    stdscr.getmaxyx lame, lamer
    lame.first
  end

  def cols
    lame, lamer = [], []
    stdscr.getmaxyx lame, lamer
    lamer.first
  end

  def curx
    lame, lamer = [], []
    stdscr.getyx lame, lamer
    lamer.first
  end
  module_function :rows, :cols, :curx
end

class BufferManager
  def initialize
    @minibuf_stack = ["foom"]
    @textfields = {}
    @in_x = ENV["TERM"] =~ /(xterm|rxvt|screen)/
    @next_color_id = 0
    @menu = nil
    @inbox = nil
  end

  def draw_screen
    draw_minibuf
    draw_status
    draw_inbox

    Ncurses.doupdate
    Ncurses.refresh
  end

  def draw_inbox
    dump = Net::HTTP.get('localhost', '/Inbox.json', 8080)
    mails = []
    @inbox = JSON.parse(dump)
    @inbox.keys.sort.each_with_index do |key, index|
      mails << Ncurses::Menu.new_item(String(index), @inbox[key][0]["Subject"])
    end
    @menu = Ncurses::Menu.new_menu mails
    @next_color_id = @next_color_id + 1
    Ncurses.init_pair @next_color_id, Ncurses::COLOR_BLACK, Ncurses::COLOR_CYAN
    Ncurses::Menu.set_menu_fore @menu, (Ncurses::COLOR_PAIR @next_color_id)
    @next_color_id = @next_color_id + 1
    Ncurses.init_pair @next_color_id, Ncurses::COLOR_WHITE, Ncurses::COLOR_BLACK
    Ncurses::Menu.set_menu_back @menu, (Ncurses::COLOR_PAIR @next_color_id)
    Ncurses::Menu.post_menu @menu
    Ncurses.refresh
  end

  def draw_thread idx
    Ncurses.clear
    key = @inbox.keys.sort[idx.to_i]
    id = @inbox[key][0]["MessageID"]
    dump = Net::HTTP.get('localhost', "/Messages/#{id}", 8080)
    Ncurses.mvaddstr 0, 0, dump
    Ncurses.refresh
  end

  def draw_minibuf
    s = @minibuf_stack[0]
    Ncurses.mvaddstr Ncurses.rows - 1, 0, s + (" " * [Ncurses.cols - s.length, 0].max)
    Ncurses.refresh
  end

  def draw_status
    s = "The glorious status"
    @next_color_id = @next_color_id + 1
    Ncurses.init_pair @next_color_id, Ncurses::COLOR_WHITE, Ncurses::COLOR_BLUE
    Ncurses.attrset (Ncurses.COLOR_PAIR @next_color_id) | Ncurses::A_BOLD
    Ncurses.mvaddstr Ncurses.rows - 2, 0, s + (" " * [Ncurses.cols - s.length, 0].max)
    Ncurses.attrset Ncurses::A_NORMAL
    Ncurses.refresh
  end

  def idle_loop
    while true
      case Ncurses.getch
      when 'q'.ord then break
      when Ncurses::KEY_DOWN then Ncurses::Menu::menu_driver @menu, Ncurses::Menu::REQ_DOWN_ITEM
      when Ncurses::KEY_UP then Ncurses::Menu::menu_driver @menu, Ncurses::Menu::REQ_UP_ITEM
      when 10 then draw_thread (Ncurses::Menu.item_name Ncurses::Menu.current_item @menu)
      end
    end
  end
end

def start_cursing
  Ncurses.initscr
  Ncurses.noecho
  Ncurses.cbreak
  Ncurses.stdscr.keypad 1
  Ncurses.use_default_colors
  Ncurses.curs_set 0
  Ncurses.start_color
end

def stop_cursing
  Ncurses.curs_set 1
  Ncurses.echo
  Ncurses.endwin
end

start_cursing
bm = BufferManager.new
bm.draw_screen
bm.idle_loop
stop_cursing
