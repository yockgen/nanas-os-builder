#!/usr/bin/env python3
"""
Madani OS Installer - QCOW2 Image Deployment Tool
A graphical installer with Calamares-style UI
"""

import gi
gi.require_version('Gtk', '3.0')
from gi.repository import Gtk, Gdk, GLib, Pango, GdkPixbuf
import subprocess
import os
import sys
import threading
import re

class InstallerWindow(Gtk.Window):
    def __init__(self):
        super().__init__(title="Madani OS Installer")
        
        # Setup logging to desktop
        self.desktop_dir = "/root/Desktop"
        os.makedirs(self.desktop_dir, exist_ok=True)
        self.log_file = os.path.join(self.desktop_dir, "installer-debug.log")
        self.log_messages = []
        self.log("Installer started")
        
        # Make window fullscreen and always on top
        self.fullscreen()
        self.set_keep_above(True)
        self.set_decorated(False)  # Remove window decorations
        self.set_modal(True)
        self.set_skip_taskbar_hint(True)
        self.set_skip_pager_hint(True)
        
        # Get screen size for proper sizing
        screen = self.get_screen()
        self.set_default_size(screen.get_width(), screen.get_height())
        
        # Installation state
        self.qcow2_image = None
        self.selected_disk = None
        self.username = ""
        self.hostname = ""
        self.password = ""
        self.current_page = 0
        
        # Find qcow2 image
        self.find_qcow2_image()
        
        # Create main layout
        main_box = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=0)
        self.add(main_box)
        
        # Sidebar
        sidebar = self.create_sidebar()
        main_box.pack_start(sidebar, False, False, 0)
        
        # Main vertical box with content and buttons (right side of window)
        main_vbox = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=0)
        main_vbox.set_hexpand(True)
        main_vbox.set_vexpand(True)
        
        # Content area with wallpaper
        content_overlay = Gtk.Overlay()
        content_overlay.get_style_context().add_class("content-area")
        
        self.content_stack = Gtk.Stack()
        self.content_stack.set_transition_type(Gtk.StackTransitionType.SLIDE_LEFT_RIGHT)
        self.content_stack.set_transition_duration(300)
        self.content_stack.set_hexpand(True)
        self.content_stack.set_vexpand(True)
        content_overlay.add(self.content_stack)
        main_vbox.pack_start(content_overlay, True, True, 0)
        
        # Navigation buttons - pinned to bottom
        button_box = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=10)
        button_box.set_margin_top(10)
        button_box.set_margin_bottom(10)
        button_box.set_margin_start(10)
        button_box.set_margin_end(10)
        # Prevent expansion and movement
        button_box.set_hexpand(True)
        button_box.set_vexpand(False)
        button_box.set_halign(Gtk.Align.FILL)
        button_box.set_valign(Gtk.Align.END)
        button_box.set_size_request(-1, 60)  # Fixed height for button area
        
        self.back_button = Gtk.Button(label="← Back")
        self.back_button.connect("clicked", self.on_back)
        self.back_button.set_sensitive(False)
        self.back_button.set_size_request(100, 40)
        button_box.pack_start(self.back_button, False, False, 0)
        
        # Cancel button
        self.cancel_button = Gtk.Button(label="Cancel")
        self.cancel_button.connect("clicked", self.on_cancel)
        self.cancel_button.get_style_context().add_class("destructive-action")
        self.cancel_button.set_size_request(100, 40)
        button_box.pack_start(self.cancel_button, False, False, 10)
        
        button_box.pack_start(Gtk.Label(), True, True, 0)  # Spacer
        
        self.next_button = Gtk.Button(label="Next →")
        self.next_button.connect("clicked", self.on_next)
        self.next_button.get_style_context().add_class("suggested-action")
        self.next_button.set_size_request(120, 40)
        button_box.pack_end(self.next_button, False, False, 0)
        
        # Pack button box at the end (bottom) with no expansion
        main_vbox.pack_end(button_box, False, False, 0)
        
        # Add main_vbox to main_box (after sidebar)
        main_box.pack_start(main_vbox, True, True, 0)
        
        # Create pages
        self.create_choice_page()
        self.create_welcome_page()
        self.create_disk_selection_page()
        self.create_user_config_page()
        self.create_summary_page()
        self.create_installation_page()
        self.create_finish_page()
        
        self.connect("destroy", Gtk.main_quit)
        
    def log(self, message):
        """Log message to both console and file"""
        timestamp = subprocess.run(["date", "+%Y-%m-%d %H:%M:%S"], capture_output=True, text=True).stdout.strip()
        log_entry = f"[{timestamp}] {message}"
        self.log_messages.append(log_entry)
        print(log_entry)
        
        # Write to file
        try:
            with open(self.log_file, "a") as f:
                f.write(log_entry + "\n")
        except Exception as e:
            print(f"Failed to write log: {e}")
    
    def on_cancel(self, widget):
        """Handle cancel button click"""
        self.log("User clicked Cancel")
        
        dialog = Gtk.MessageDialog(
            transient_for=self,
            flags=0,
            message_type=Gtk.MessageType.QUESTION,
            buttons=Gtk.ButtonsType.YES_NO,
            text="Exit Installer?"
        )
        dialog.format_secondary_text(
            f"Are you sure you want to exit the installer?\n\nDebug log saved to: {self.log_file}"
        )
        response = dialog.run()
        dialog.destroy()
        
        if response == Gtk.ResponseType.YES:
            self.log("User confirmed exit")
            self.save_final_log()
            Gtk.main_quit()
    
    def save_final_log(self):
        """Save final log with system information"""
        try:
            with open(self.log_file, "a") as f:
                f.write("\n=== SYSTEM INFORMATION ===\n")
                
                # Mount info
                result = subprocess.run(["mount"], capture_output=True, text=True)
                f.write("\nMounts:\n" + result.stdout + "\n")
                
                # Block devices
                result = subprocess.run(["lsblk", "-o", "NAME,SIZE,TYPE,MOUNTPOINT,FSTYPE"], capture_output=True, text=True)
                f.write("\nBlock Devices:\n" + result.stdout + "\n")
                
                # Directory listing
                for path in ["/cdrom", "/cdrom/images"]:
                    if os.path.isdir(path):
                        result = subprocess.run(["ls", "-la", path], capture_output=True, text=True)
                        f.write(f"\nContents of {path}:\n" + result.stdout + "\n")
                
                f.write("\n=== END OF LOG ===\n")
        except Exception as e:
            print(f"Failed to save final log: {e}")
    
    def create_sidebar(self):
        sidebar = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=0)
        sidebar.set_size_request(250, -1)
        sidebar.get_style_context().add_class("sidebar")
        
        # Apply custom CSS
        css = b"""
        .sidebar {
            background: linear-gradient(to bottom, #2c3e50, #34495e);
            color: white;
            padding: 20px;
        }
        .sidebar-item {
            padding: 15px;
            margin: 5px 0;
            border-radius: 5px;
            color: #bdc3c7;
        }
        .sidebar-item-active {
            background-color: rgba(52, 152, 219, 0.3);
            color: white;
            font-weight: bold;
        }
        .sidebar-item-completed {
            color: #27ae60;
        }
        .content-area {
            background-color: #f5f6fa;
        }
        .content-overlay {
            background-color: rgba(255, 255, 255, 0.92);
            border-radius: 10px;
        }
        .page-box {
            background-color: rgba(255, 255, 255, 0.85);
            border-radius: 8px;
            padding: 20px;
        }
        """
        
        css_provider = Gtk.CssProvider()
        css_provider.load_from_data(css)
        Gtk.StyleContext.add_provider_for_screen(
            Gdk.Screen.get_default(),
            css_provider,
            Gtk.STYLE_PROVIDER_PRIORITY_APPLICATION
        )
        
        # Logo/Title
        logo_path = "/usr/share/madani/mos-logo.png"
        if os.path.exists(logo_path):
            pixbuf = GdkPixbuf.Pixbuf.new_from_file_at_scale(logo_path, 240, 240, True)
            logo = Gtk.Image.new_from_pixbuf(pixbuf)
        else:
            logo = Gtk.Image.new_from_icon_name("distributor-logo", Gtk.IconSize.DIALOG)
            logo.set_pixel_size(240)
        logo.set_margin_bottom(10)
        sidebar.pack_start(logo, False, False, 0)
        
        title = Gtk.Label()
        title.set_markup("<span size='large' weight='bold'>Madani OS</span>")
        title.set_margin_bottom(30)
        sidebar.pack_start(title, False, False, 0)
        
        # Steps
        self.sidebar_items = []
        steps = [
            ("Start", "system-run-symbolic"),
            ("Welcome", "dialog-information-symbolic"),
            ("Select Disk", "drive-harddisk-symbolic"),
            ("User Setup", "system-users-symbolic"),
            ("Summary", "emblem-documents-symbolic"),
            ("Installation", "system-software-install-symbolic"),
            ("Finish", "emblem-default-symbolic")
        ]
        
        for i, (label, icon_name) in enumerate(steps):
            item_box = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=10)
            item_box.get_style_context().add_class("sidebar-item")
            
            icon = Gtk.Image.new_from_icon_name(icon_name, Gtk.IconSize.BUTTON)
            item_box.pack_start(icon, False, False, 0)
            
            label_widget = Gtk.Label(label=label)
            label_widget.set_halign(Gtk.Align.START)
            item_box.pack_start(label_widget, True, True, 0)
            
            sidebar.pack_start(item_box, False, False, 0)
            self.sidebar_items.append(item_box)
        
        # Update first item as active
        self.update_sidebar(0)
        
        return sidebar
    
    def update_sidebar(self, current_page):
        for i, item in enumerate(self.sidebar_items):
            item.get_style_context().remove_class("sidebar-item-active")
            item.get_style_context().remove_class("sidebar-item-completed")
            
            if i == current_page:
                item.get_style_context().add_class("sidebar-item-active")
            elif i < current_page:
                item.get_style_context().add_class("sidebar-item-completed")
    
    def find_qcow2_image(self):
        search_paths = [
            "/cdrom/images",
            "/cdrom",
            "/run/archiso/bootmnt/images",
            "/run/archiso/bootmnt",
            "/mnt/cdrom/images",
            "/mnt/cdrom",
            "/media/cdrom/images",
            "/media/cdrom"
        ]
        
        self.log("Searching for QCOW2 images...")
        for path in search_paths:
            self.log(f"Checking path: {path}")
            if os.path.isdir(path):
                self.log(f"Directory exists: {path}")
                try:
                    # List directory contents for debugging
                    try:
                        contents = os.listdir(path)
                        self.log(f"Contents of {path}: {contents}")
                    except Exception as e:
                        self.log(f"Cannot list {path}: {e}")
                    
                    result = subprocess.run(
                        ["find", path, "-maxdepth", "3", "-name", "*.qcow2", "-type", "f"],
                        capture_output=True, text=True
                    )
                    images = result.stdout.strip().split('\n')
                    if images and images[0]:
                        self.qcow2_image = images[0]
                        self.log(f"Found QCOW2 image: {self.qcow2_image}")
                        return
                except Exception as e:
                    self.log(f"Error searching {path}: {e}")
            else:
                self.log(f"Directory does not exist: {path}")
        
        self.log("WARNING: No QCOW2 image found!")
    
    def create_choice_page(self):
        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=30)
        box.set_margin_top(50)
        box.set_margin_bottom(50)
        box.set_margin_start(50)
        box.set_margin_end(50)
        box.set_halign(Gtk.Align.CENTER)
        box.set_valign(Gtk.Align.CENTER)
        
        # Load Madani OS logo
        logo_path = "/usr/share/madani/mos-logo.png"
        if os.path.exists(logo_path):
            pixbuf = GdkPixbuf.Pixbuf.new_from_file_at_scale(logo_path, 512, 512, True)
            icon = Gtk.Image.new_from_pixbuf(pixbuf)
        else:
            icon = Gtk.Image.new_from_icon_name("distributor-logo", Gtk.IconSize.DIALOG)
            icon.set_pixel_size(512)
        box.pack_start(icon, False, False, 0)
        
        title = Gtk.Label()
        title.set_markup("<span size='xx-large' weight='bold'>Welcome to Madani OS</span>")
        box.pack_start(title, False, False, 0)
        
        desc = Gtk.Label()
        desc.set_markup(
            "You can try Madani OS without making any changes to your computer,\n"
            "or install it permanently."
        )
        desc.set_line_wrap(True)
        desc.set_justify(Gtk.Justification.CENTER)
        box.pack_start(desc, False, False, 20)
        
        # Buttons container
        button_container = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=20)
        button_container.set_halign(Gtk.Align.CENTER)
        
        # Try button
        try_button = Gtk.Button()
        try_box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=10)
        try_box.set_margin_top(20)
        try_box.set_margin_bottom(20)
        try_box.set_margin_start(30)
        try_box.set_margin_end(30)
        try_icon = Gtk.Image.new_from_icon_name("media-playback-start", Gtk.IconSize.DIALOG)
        try_icon.set_pixel_size(64)
        try_box.pack_start(try_icon, False, False, 0)
        try_label = Gtk.Label()
        try_label.set_markup("<span size='large' weight='bold'>Try Madani OS</span>")
        try_box.pack_start(try_label, False, False, 0)
        try_desc = Gtk.Label()
        try_desc.set_markup("<small>Boot into live environment\nwithout installation</small>")
        try_desc.set_justify(Gtk.Justification.CENTER)
        try_box.pack_start(try_desc, False, False, 0)
        try_button.add(try_box)
        try_button.connect("clicked", self.on_try_clicked)
        button_container.pack_start(try_button, False, False, 0)
        
        # Install button
        install_button = Gtk.Button()
        install_box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=10)
        install_box.set_margin_top(20)
        install_box.set_margin_bottom(20)
        install_box.set_margin_start(30)
        install_box.set_margin_end(30)
        install_icon = Gtk.Image.new_from_icon_name("system-software-install", Gtk.IconSize.DIALOG)
        install_icon.set_pixel_size(64)
        install_box.pack_start(install_icon, False, False, 0)
        install_label = Gtk.Label()
        install_label.set_markup("<span size='large' weight='bold'>Install Madani OS</span>")
        install_box.pack_start(install_label, False, False, 0)
        install_desc = Gtk.Label()
        install_desc.set_markup("<small>Install Madani OS\npermanently to your disk</small>")
        install_desc.set_justify(Gtk.Justification.CENTER)
        install_box.pack_start(install_desc, False, False, 0)
        install_button.add(install_box)
        install_button.connect("clicked", self.on_install_clicked)
        install_button.get_style_context().add_class("suggested-action")
        button_container.pack_start(install_button, False, False, 0)
        
        box.pack_start(button_container, False, False, 0)
        
        self.content_stack.add_titled(box, "choice", "Choice")
    
    def on_try_clicked(self, widget):
        self.log("User selected Try - exiting installer")
        self.save_final_log()
        Gtk.main_quit()
    
    def on_install_clicked(self, widget):
        self.log("User selected Install - continuing to installation")
        self.current_page = 1
        self.show_page(1)
    
    def create_welcome_page(self):
        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=20)
        box.set_margin_top(50)
        box.set_margin_bottom(50)
        box.set_margin_start(50)
        box.set_margin_end(50)
        box.set_halign(Gtk.Align.CENTER)
        box.set_valign(Gtk.Align.CENTER)
        
        # Load Madani OS logo
        logo_path = "/usr/share/madani/mos-logo.png"
        if os.path.exists(logo_path):
            pixbuf = GdkPixbuf.Pixbuf.new_from_file_at_scale(logo_path, 512, 512, True)
            icon = Gtk.Image.new_from_pixbuf(pixbuf)
        else:
            icon = Gtk.Image.new_from_icon_name("distributor-logo", Gtk.IconSize.DIALOG)
            icon.set_pixel_size(512)
        box.pack_start(icon, False, False, 0)
        
        title = Gtk.Label()
        title.set_markup("<span size='xx-large' weight='bold'>Welcome to Madani OS Installer</span>")
        box.pack_start(title, False, False, 0)
        
        desc = Gtk.Label()
        desc.set_markup(
            "This installer will guide you through the installation process.\n\n"
            "The installation will deploy a pre-configured system image to your disk.\n"
            "All data on the selected disk will be erased."
        )
        desc.set_line_wrap(True)
        desc.set_justify(Gtk.Justification.CENTER)
        box.pack_start(desc, False, False, 0)
        
        if self.qcow2_image:
            image_info = Gtk.Label()
            image_info.set_markup(f"<b>Image:</b> {os.path.basename(self.qcow2_image)}")
            box.pack_start(image_info, False, False, 20)
        else:
            error = Gtk.Label()
            error.set_markup("<span color='red' weight='bold'>⚠ No installation image found!</span>")
            box.pack_start(error, False, False, 20)
        
        self.content_stack.add_titled(box, "welcome", "Welcome")
    
    def create_disk_selection_page(self):
        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=20)
        box.set_margin_top(30)
        box.set_margin_bottom(30)
        box.set_margin_start(40)
        box.set_margin_end(40)
        
        title = Gtk.Label()
        title.set_markup("<span size='large' weight='bold'>Select Installation Disk</span>")
        title.set_halign(Gtk.Align.START)
        box.pack_start(title, False, False, 0)
        
        warning = Gtk.Label()
        warning.set_markup(
            "<span color='#e74c3c' weight='bold'>⚠ WARNING:</span> "
            "All data on the selected disk will be permanently erased!"
        )
        warning.set_halign(Gtk.Align.START)
        box.pack_start(warning, False, False, 0)
        
        # Disk list
        scrolled = Gtk.ScrolledWindow()
        scrolled.set_vexpand(True)
        scrolled.set_policy(Gtk.PolicyType.NEVER, Gtk.PolicyType.AUTOMATIC)
        
        self.disk_store = Gtk.ListStore(str, str, str, str)  # device, size, model, full_path
        self.disk_view = Gtk.TreeView(model=self.disk_store)
        self.disk_view.set_headers_visible(True)
        
        # Columns
        renderer = Gtk.CellRendererText()
        column = Gtk.TreeViewColumn("Device", renderer, text=0)
        self.disk_view.append_column(column)
        
        column = Gtk.TreeViewColumn("Size", renderer, text=1)
        self.disk_view.append_column(column)
        
        column = Gtk.TreeViewColumn("Model", renderer, text=2)
        self.disk_view.append_column(column)
        
        scrolled.add(self.disk_view)
        box.pack_start(scrolled, True, True, 0)
        
        # Refresh button
        refresh_btn = Gtk.Button(label="🔄 Refresh Disk List")
        refresh_btn.connect("clicked", self.refresh_disks)
        box.pack_start(refresh_btn, False, False, 0)
        
        self.content_stack.add_titled(box, "disk", "Disk Selection")
        self.refresh_disks()
    
    def refresh_disks(self, widget=None):
        self.disk_store.clear()
        try:
            result = subprocess.run(
                ["lsblk", "-dpno", "NAME,SIZE,MODEL"],
                capture_output=True, text=True
            )
            for line in result.stdout.strip().split('\n'):
                if line and not any(x in line for x in ['loop', 'sr']):
                    parts = line.split(None, 2)
                    if len(parts) >= 2:
                        device = parts[0]
                        size = parts[1] if len(parts) > 1 else "Unknown"
                        model = parts[2] if len(parts) > 2 else "Unknown"
                        self.disk_store.append([device, size, model, device])
        except Exception as e:
            print(f"Error listing disks: {e}")
    
    def create_user_config_page(self):
        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=20)
        box.set_margin_top(30)
        box.set_margin_bottom(30)
        box.set_margin_start(40)
        box.set_margin_end(40)
        
        title = Gtk.Label()
        title.set_markup("<span size='large' weight='bold'>User Configuration</span>")
        title.set_halign(Gtk.Align.START)
        box.pack_start(title, False, False, 0)
        
        # Form
        grid = Gtk.Grid()
        grid.set_column_spacing(10)
        grid.set_row_spacing(15)
        
        # Username
        label = Gtk.Label(label="Username:")
        label.set_halign(Gtk.Align.END)
        grid.attach(label, 0, 0, 1, 1)
        
        self.username_entry = Gtk.Entry()
        self.username_entry.set_placeholder_text("Enter username")
        self.username_entry.set_hexpand(True)
        grid.attach(self.username_entry, 1, 0, 1, 1)
        
        # Computer Name
        label = Gtk.Label(label="Computer Name:")
        label.set_halign(Gtk.Align.END)
        grid.attach(label, 0, 1, 1, 1)
        
        self.hostname_entry = Gtk.Entry()
        self.hostname_entry.set_placeholder_text("Enter computer name")
        grid.attach(self.hostname_entry, 1, 1, 1, 1)
        
        # Password
        label = Gtk.Label(label="Password:")
        label.set_halign(Gtk.Align.END)
        grid.attach(label, 0, 2, 1, 1)
        
        self.password_entry = Gtk.Entry()
        self.password_entry.set_visibility(False)
        self.password_entry.set_placeholder_text("Enter password")
        grid.attach(self.password_entry, 1, 2, 1, 1)
        
        # Confirm Password
        label = Gtk.Label(label="Confirm Password:")
        label.set_halign(Gtk.Align.END)
        grid.attach(label, 0, 3, 1, 1)
        
        self.password_confirm_entry = Gtk.Entry()
        self.password_confirm_entry.set_visibility(False)
        self.password_confirm_entry.set_placeholder_text("Re-enter password")
        grid.attach(self.password_confirm_entry, 1, 3, 1, 1)
        
        box.pack_start(grid, False, False, 20)
        
        note = Gtk.Label()
        note.set_markup(
            "<small><i>Note: These settings will be configured after the system image is deployed.</i></small>"
        )
        note.set_halign(Gtk.Align.START)
        box.pack_start(note, False, False, 0)
        
        self.content_stack.add_titled(box, "user", "User Setup")
    
    def create_summary_page(self):
        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=20)
        box.set_margin_top(30)
        box.set_margin_bottom(30)
        box.set_margin_start(40)
        box.set_margin_end(40)
        
        title = Gtk.Label()
        title.set_markup("<span size='large' weight='bold'>Installation Summary</span>")
        title.set_halign(Gtk.Align.START)
        box.pack_start(title, False, False, 0)
        
        self.summary_text = Gtk.Label()
        self.summary_text.set_halign(Gtk.Align.START)
        self.summary_text.set_line_wrap(True)
        box.pack_start(self.summary_text, False, False, 0)
        
        warning = Gtk.Label()
        warning.set_markup(
            "\n<span color='#e74c3c' weight='bold' size='large'>⚠ FINAL WARNING ⚠</span>\n\n"
            "Clicking 'Install' will begin the installation process.\n"
            "All data on the selected disk will be permanently erased.\n"
            "This action cannot be undone!"
        )
        warning.set_line_wrap(True)
        warning.set_justify(Gtk.Justification.CENTER)
        box.pack_start(warning, True, True, 20)
        
        self.content_stack.add_titled(box, "summary", "Summary")
    
    def create_installation_page(self):
        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=20)
        box.set_margin_top(50)
        box.set_margin_bottom(50)
        box.set_margin_start(50)
        box.set_margin_end(50)
        box.set_valign(Gtk.Align.CENTER)
        
        title = Gtk.Label()
        title.set_markup("<span size='large' weight='bold'>Installing System</span>")
        box.pack_start(title, False, False, 0)
        
        self.install_status = Gtk.Label(label="Preparing installation...")
        box.pack_start(self.install_status, False, False, 0)
        
        self.install_progress = Gtk.ProgressBar()
        self.install_progress.set_show_text(True)
        box.pack_start(self.install_progress, False, False, 0)
        
        self.content_stack.add_titled(box, "install", "Installing")
    
    def create_finish_page(self):
        box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=20)
        box.set_margin_top(50)
        box.set_margin_bottom(50)
        box.set_margin_start(50)
        box.set_margin_end(50)
        box.set_halign(Gtk.Align.CENTER)
        box.set_valign(Gtk.Align.CENTER)
        
        # Success icon or logo
        logo_path = "/usr/share/madani/mos-logo.png"
        if os.path.exists(logo_path):
            pixbuf = GdkPixbuf.Pixbuf.new_from_file_at_scale(logo_path, 512, 512, True)
            icon = Gtk.Image.new_from_pixbuf(pixbuf)
        else:
            icon = Gtk.Image.new_from_icon_name("emblem-default", Gtk.IconSize.DIALOG)
            icon.set_pixel_size(512)
        box.pack_start(icon, False, False, 0)
        
        title = Gtk.Label()
        title.set_markup("<span size='xx-large' weight='bold' color='#27ae60'>Installation Complete!</span>")
        box.pack_start(title, False, False, 0)
        
        message = Gtk.Label()
        message.set_markup(
            "The system has been successfully installed.\n\n"
            "Please remove the installation media and reboot your computer.\n"
            "The system will boot from the installed disk."
        )
        message.set_line_wrap(True)
        message.set_justify(Gtk.Justification.CENTER)
        box.pack_start(message, False, False, 0)
        
        self.content_stack.add_titled(box, "finish", "Finish")
    
    def on_back(self, widget):
        if self.current_page > 1:  # Can't go back past welcome page (page 1)
            self.current_page -= 1
            self.show_page(self.current_page)
    
    def on_next(self, widget):
        # Validate current page
        if not self.validate_current_page():
            return
        
        if self.current_page < 6:
            self.current_page += 1
            
            if self.current_page == 4:  # Summary page
                self.update_summary()
            
            self.show_page(self.current_page)
            
            if self.current_page == 5:  # Installation page
                self.next_button.set_sensitive(False)
                self.back_button.set_sensitive(False)
                # Start installation after showing the page
                GLib.timeout_add(500, self.perform_installation)
                return
        elif self.current_page == 6:  # Finish page
            # Reboot
            subprocess.run(["reboot"])
    
    def show_page(self, page):
        pages = ["choice", "welcome", "disk", "user", "summary", "install", "finish"]
        self.content_stack.set_visible_child_name(pages[page])
        self.update_sidebar(page)
        
        # Update buttons - hide navigation buttons on choice page
        if page == 0:  # Choice page
            self.back_button.set_visible(False)
            self.next_button.set_visible(False)
            self.cancel_button.set_visible(False)
        else:
            self.back_button.set_visible(True)
            self.next_button.set_visible(True)
            self.cancel_button.set_visible(True)
            self.back_button.set_sensitive(page > 1 and page < 5)
            
            if page == 1:
                self.next_button.set_label("Get Started →")
            elif page == 4:
                self.next_button.set_label("Install Now")
            elif page == 6:
                self.next_button.set_label("Reboot")
            else:
                self.next_button.set_label("Next →")
            
            if page == 5:  # Installing
                self.next_button.set_visible(False)
    
    def validate_current_page(self):
        if self.current_page == 2:  # Disk selection
            selection = self.disk_view.get_selection()
            model, treeiter = selection.get_selected()
            if treeiter is None:
                dialog = Gtk.MessageDialog(
                    transient_for=self,
                    flags=0,
                    message_type=Gtk.MessageType.ERROR,
                    buttons=Gtk.ButtonsType.OK,
                    text="Please select a disk"
                )
                dialog.run()
                dialog.destroy()
                return False
            self.selected_disk = model[treeiter][3]
        
        elif self.current_page == 3:  # User config
            self.username = self.username_entry.get_text().strip()
            self.hostname = self.hostname_entry.get_text().strip()
            self.password = self.password_entry.get_text()
            password_confirm = self.password_confirm_entry.get_text()
            
            if not self.username:
                self.show_error("Please enter a username")
                return False
            
            if not self.hostname:
                self.show_error("Please enter a computer name")
                return False
            
            if not self.password:
                self.show_error("Please enter a password")
                return False
            
            if self.password != password_confirm:
                self.show_error("Passwords do not match")
                return False
        
        return True
    
    def update_summary(self):
        summary = f"""<b>Installation Target:</b> {self.selected_disk}

<b>User Configuration:</b>
  • Username: {self.username}
  • Computer Name: {self.hostname}
  • Password: {'•' * len(self.password)}

<b>Image:</b> {os.path.basename(self.qcow2_image) if self.qcow2_image else 'N/A'}
"""
        self.summary_text.set_markup(summary)
    
    def show_error(self, message):
        dialog = Gtk.MessageDialog(
            transient_for=self,
            flags=0,
            message_type=Gtk.MessageType.ERROR,
            buttons=Gtk.ButtonsType.OK,
            text=message
        )
        dialog.run()
        dialog.destroy()
    
    def perform_installation(self):
        def install_thread():
            try:
                # Unmount partitions
                GLib.idle_add(self.update_install_status, "Unmounting partitions...", 0.1)
                subprocess.run(f"for part in $(lsblk -ln -o NAME {self.selected_disk} 2>/dev/null | grep -v '^$(basename {self.selected_disk})$' || true); do umount -f /dev/$part 2>/dev/null || true; done", shell=True)
                
                # Verify image
                GLib.idle_add(self.update_install_status, "Verifying image...", 0.2)
                subprocess.run(["qemu-img", "info", self.qcow2_image], check=True, capture_output=True)
                
                # Flash image
                GLib.idle_add(self.update_install_status, "Flashing image to disk...", 0.3)
                subprocess.run(
                    ["qemu-img", "convert", "-f", "qcow2", "-O", "raw", "-p", self.qcow2_image, self.selected_disk],
                    check=True
                )
                
                # Sync
                GLib.idle_add(self.update_install_status, "Syncing filesystem...", 0.95)
                subprocess.run(["sync"])
                
                # Reload partition table
                subprocess.run(["partprobe", self.selected_disk], capture_output=True)
                import time
                time.sleep(2)
                
                # Configure the installed system
                GLib.idle_add(self.update_install_status, "Configuring system...", 0.96)
                self.configure_installed_system()
                
                GLib.idle_add(self.update_install_status, "Installation complete!", 1.0)
                GLib.idle_add(self.installation_complete)
                
            except Exception as e:
                GLib.idle_add(self.installation_failed, str(e))
        
        thread = threading.Thread(target=install_thread)
        thread.daemon = True
        thread.start()
        return False  # Don't repeat timeout
    
    def configure_installed_system(self):
        """Configure username, hostname, and password on the installed system"""
        mount_point = "/mnt/madani-install"
        
        try:
            # Create mount point
            os.makedirs(mount_point, exist_ok=True)
            
            # Find and mount the root partition (usually partition 2)
            result = subprocess.run(
                ["lsblk", "-ln", "-o", "NAME,FSTYPE,MOUNTPOINT", self.selected_disk],
                capture_output=True, text=True
            )
            
            root_partition = None
            for line in result.stdout.strip().split('\n'):
                parts = line.split()
                if len(parts) >= 2 and parts[1] in ['ext4', 'xfs', 'btrfs']:
                    # Get the full device path
                    part_name = parts[0]
                    if not part_name.startswith('/dev/'):
                        if 'nvme' in self.selected_disk or 'mmcblk' in self.selected_disk:
                            root_partition = f"/dev/{part_name}"
                        else:
                            root_partition = f"/dev/{part_name}"
                    else:
                        root_partition = part_name
                    break
            
            if not root_partition:
                self.log("Could not find root partition, trying default...")
                # Try default partition naming
                if 'nvme' in self.selected_disk or 'mmcblk' in self.selected_disk:
                    root_partition = f"{self.selected_disk}p2"
                else:
                    root_partition = f"{self.selected_disk}2"
            
            self.log(f"Mounting root partition: {root_partition}")
            subprocess.run(["mount", root_partition, mount_point], check=True)
            
            # Set hostname
            self.log(f"Setting hostname to: {self.hostname}")
            with open(f"{mount_point}/etc/hostname", "w") as f:
                f.write(f"{self.hostname}\n")
            
            # Update /etc/hosts
            hosts_content = f"""127.0.0.1	localhost
127.0.1.1	{self.hostname}

::1		localhost ip6-localhost ip6-loopback
ff02::1		ip6-allnodes
ff02::2		ip6-allrouters
"""
            with open(f"{mount_point}/etc/hosts", "w") as f:
                f.write(hosts_content)
            
            # Create user and set password using chroot
            self.log(f"Creating user: {self.username}")
            
            # Mount necessary filesystems for chroot
            subprocess.run(["mount", "-t", "proc", "proc", f"{mount_point}/proc"], check=False)
            subprocess.run(["mount", "-t", "sysfs", "sys", f"{mount_point}/sys"], check=False)
            subprocess.run(["mount", "-o", "bind", "/dev", f"{mount_point}/dev"], check=False)
            subprocess.run(["mount", "-t", "devpts", "devpts", f"{mount_point}/dev/pts"], check=False)
            
            # Create user with home directory
            subprocess.run(
                ["chroot", mount_point, "useradd", "-m", "-s", "/bin/bash", self.username],
                check=True
            )
            
            # Set user password
            subprocess.run(
                ["chroot", mount_point, "bash", "-c", f"echo '{self.username}:{self.password}' | chpasswd"],
                check=True
            )
            
            # Add user to sudo group
            subprocess.run(
                ["chroot", mount_point, "usermod", "-aG", "sudo", self.username],
                check=False  # sudo group might not exist
            )
            
            # Set root password to the same
            subprocess.run(
                ["chroot", mount_point, "bash", "-c", f"echo 'root:{self.password}' | chpasswd"],
                check=True
            )
            
            self.log("System configuration completed successfully")
            
        except Exception as e:
            self.log(f"Error configuring system: {e}")
            import traceback
            self.log(traceback.format_exc())
        
        finally:
            # Unmount everything
            self.log("Unmounting filesystems...")
            subprocess.run(["umount", "-f", f"{mount_point}/dev/pts"], check=False)
            subprocess.run(["umount", "-f", f"{mount_point}/dev"], check=False)
            subprocess.run(["umount", "-f", f"{mount_point}/sys"], check=False)
            subprocess.run(["umount", "-f", f"{mount_point}/proc"], check=False)
            subprocess.run(["umount", "-f", mount_point], check=False)
            subprocess.run(["sync"])
    
    def update_install_status(self, status, progress):
        self.install_status.set_text(status)
        self.install_progress.set_fraction(progress)
        return False
    
    def installation_complete(self):
        self.current_page = 6
        self.show_page(6)
        self.next_button.set_visible(True)
        self.next_button.set_sensitive(True)
        return False
    
    def installation_failed(self, error):
        dialog = Gtk.MessageDialog(
            transient_for=self,
            flags=0,
            message_type=Gtk.MessageType.ERROR,
            buttons=Gtk.ButtonsType.OK,
            text="Installation Failed"
        )
        dialog.format_secondary_text(f"Error: {error}")
        dialog.run()
        dialog.destroy()
        self.current_page = 1
        self.show_page(1)
        self.back_button.set_sensitive(False)
        self.next_button.set_sensitive(True)
        return False

if __name__ == "__main__":
    # Check if installer has already been run this session
    flag_file = "/tmp/.madani-installer-run"
    
    if os.path.exists(flag_file):
        print("[INSTALLER] Installer already launched this session. Exiting.")
        sys.exit(0)
    
    # Create flag file to prevent multiple instances
    try:
        with open(flag_file, "w") as f:
            f.write("Installer launched\n")
        print("[INSTALLER] Created flag file to prevent duplicate launches")
    except Exception as e:
        print(f"[INSTALLER] Warning: Could not create flag file: {e}")
    
    # Wait for /cdrom to be mounted (up to 30 seconds)
    print("[INSTALLER] Waiting for /cdrom to be mounted...")
    max_wait = 30
    for i in range(max_wait):
        if os.path.ismount("/cdrom") or os.path.exists("/cdrom/images"):
            print(f"[INSTALLER] /cdrom is ready after {i} seconds")
            break
        print(f"[INSTALLER] Waiting for /cdrom mount... ({i+1}/{max_wait})")
        import time
        time.sleep(1)
    else:
        print("[INSTALLER] WARNING: /cdrom not mounted after 30 seconds, continuing anyway...")
    
    # Redirect stderr to stdout to see any errors
    sys.stderr = sys.stdout
    
    win = InstallerWindow()
    win.show_all()
    Gtk.main()
