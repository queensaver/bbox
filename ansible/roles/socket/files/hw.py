#!/usr/bin/env python3
"""Hardware module hw.py

Interfacing and controlling hardware.
"""

import time
import math
import threading
import logging
import argparse
from distutils import util
from ast import literal_eval
from typing import Callable
import os
import logging
from abc import ABC, abstractmethod
from typing import Tuple, Type


class HardwareInterface(ABC):
    """Abstract hardware interfacing class.
    
    Interfaces the hardware to perform all actions requiring hardware.
    """
    
    steps_per_rev = 200
    """Stepper motors number of steps per revolution
    """
    
    @abstractmethod
    def set_light_gpio(self, value: bool):
        """Set the level of the gpio controlling the light.
        
        Parameters
        ---------
        value
            Value to set light to
        """
        
        pass

    @abstractmethod
    def get_light_gpio(self) -> bool:
        """Get the level of the gpio controlling the light.
        
        RETURNS
        -------
        bool
            State of the light controlling gpio: True (light), False (no light).
        """
        
        pass
        
    @abstractmethod
    def set_stepper_motor(self, steps: int, ts: float):
        """Perform synchronously given steps with stepper motor.
        
        PARAMETERS
        ----------
        steps
            Steps for the stepper motor to perform.
        ts
            Sleep time between steps.
        """
        
        pass

    def set_stepper_motor_async(self, steps: int, ts: float):
        """Perform asynchronously given steps with stepper motor.
        
        PARAMETERS
        ----------
        steps
            Steps for the stepper motor to perform.
        ts
            Sleep time between steps.
        """
        
        threading.Thread(target=self.set_stepper_motor, args=(steps,)).start()

    @abstractmethod
    def get_stepper_motor_stat(self) -> Tuple[bool, bool, bool]:
        """Get status of stepper motor.
        
        RETURNS
        -------
        Tuple[bool, bool]
            State of stepper motor: enable, step, direction.
        """
        
        pass
    

class HardwareInterfaceDummy(HardwareInterface):
    """Dummy hardware interface for platforms without gpio access.
    """
    
    def __init__(self, *args, **kwargs):
        super(HardwareInterfaceDummy, self).__init__()
        self._gpio_light = False
        self._gpio_stepper_motor = False
    
    def set_light_gpio(self, value):
        self._gpio_light = value
    
    def get_light_gpio(self):
        return self._gpio_light
    
    def set_stepper_motor(self, steps, ts):
        self._gpio_stepper_motor = True
        time.sleep(steps * ts)
        self._gpio_stepper_motor = False
        
    def set_stepper_motor_async(self, steps, ts):
        threading.Thread(target=self.set_stepper_motor, args=(steps, ts,)).start()
        
    def get_stepper_motor_stat(self):
        return self._gpio_stepper_motor, True


class HardwareInterfacePi(HardwareInterface):
    """Hardware interface for RaspberryPi.
    
    Parameters
    ----------
    pin_light
        GPIO pin (board numbering) to control the light
    pin_n_en
        GPIO pin (board numbering) to enable the motor (negated)
    pin_n_reset
        GPIO pin (board numbering) to reset driver (negated)
    pin_n_sleep
        GPIO pin (board numbering) to set motor driver into sleep mode (negated)
    pin_step
        GPIO pin (board numbering) to perform a motor step
    pin_dir
        GPIO pin (board numbering) to control the motor direction
    mot_steps_per_rev
        Motor steps per revolution
    """
    
    pulse_dur_min = 5.0e-6
    """Minimum pulse duration for stepper motor control
    """
    
    def __init__(self, pin_light: int, pin_n_en: int, pin_n_reset: int, pin_n_sleep: int, pin_step: int, pin_dir: int,
                 mot_steps_per_rev: int):
        # from beeclient.drop_scanner.image import ImageProvider
        try:
            import RPi.GPIO as GPIO
            self._gpio = GPIO
        except RuntimeError:
            raise RuntimeError("Error importing RPi.GPIO! Have you added your user to the group gpio?")
        
        self._pin_light = pin_light
        self._pin_n_en = pin_n_en
        self._pin_n_slp = pin_n_sleep
        self._pin_n_rst = pin_n_reset
        self._pin_step = pin_step
        self._pin_dir = pin_dir
        self.steps_per_rev = mot_steps_per_rev
        self._gpio.setmode(self._gpio.BOARD)
        self._gpio.setwarnings(False)
        self._gpio.setup([self._pin_dir, self._pin_step, self._pin_light, self._pin_n_slp],
                         self._gpio.OUT, initial=self._gpio.LOW)
        self._gpio.setup([self._pin_n_en, self._pin_n_rst],
                         self._gpio.OUT, initial=self._gpio.HIGH)
        
        # self.image_provider_class = ImageProvider

        super(HardwareInterfacePi, self).__init__()

    def set_light_gpio(self, value):
        if value:
            self._gpio.output(self._pin_light, self._gpio.HIGH)
        else:
            self._gpio.output(self._pin_light, self._gpio.LOW)

    def get_light_gpio(self):
        return self._gpio.input(self._pin_light)

    @staticmethod
    def tmc2209_velocity_scaling_compensation(v):
        return v / (-6.686362 * v**2 + 0.955385 * v + 0.000352)
    
    def set_stepper_motor(self, steps, ts):
        if ts < self.pulse_dur_min:
            ts = self.pulse_dur_min
        
        try:
            logging.debug('init pwm')
            
            # stepper pwm
            pwm = self._gpio.PWM(self._pin_step, int(1 / ts))

            # disable motor driver deep sleep
            logging.debug('n_slp: {} -> {}'.format(self._pin_n_slp, 'HIGH'))
            self._gpio.output(self._pin_n_slp, self._gpio.HIGH)
            time.sleep(0.005)
            
            # enable motor driver outputs
            logging.debug('n_en:  {} -> {}'.format(self._pin_n_en, 'LOW'))
            self._gpio.output(self._pin_n_en, self._gpio.LOW)
            time.sleep(0.005)
            
            # wait a bit
            time.sleep(0.1)

            # perform movement
            logging.debug('pwm:   {} -> {}'.format(self._pin_step, 'ON'))
            pwm.start(50)  # pwm with 50% duty cycle
            time.sleep(ts * steps)
            logging.debug('pwm:   {} -> {}'.format(self._pin_step, 'OFF'))
            pwm.stop()
                
        except KeyboardInterrupt:
            pass
        
        finally:
            # disable motor
            logging.debug('disable motor')
            logging.debug('n_en:  {} -> {}'.format(self._pin_n_en, 'HIGH'))
            self._gpio.output(self._pin_n_en, self._gpio.HIGH)
            # reset pin states
            logging.debug('n_slp: {} -> {}'.format(self._pin_n_slp, 'LOW'))
            self._gpio.output(self._pin_n_slp, self._gpio.LOW)
            logging.debug('stp:   {} -> {}'.format(self._pin_step, 'LOW'))
            self._gpio.output(self._pin_step, self._gpio.LOW)

    def get_stepper_motor_stat(self):
        return self._gpio.input(self._pin_n_en), self._gpio.input(self._pin_step), self._gpio.input(self._pin_dir)


class HardwareAbstraction(object):
    """An hardware abstraction class implemented as Singleton
    """
    _instance = None
    
    def __new__(cls, hw_interface_class: Type[HardwareInterface], *args, **kwargs) -> 'HardwareAbstraction':
        """Instantiate hardware abstraction singleton
        
        Parameters
        ----------
        hw_interface_class
            Hardware interface to instantiate
        
        Returns
        -------
        HardwareAbstraction
            Hardware abstraction instance
        """
        
        if cls._instance is None:
            cls._instance = super(HardwareAbstraction, cls).__new__(cls)
            cls._instance._init(hw_interface_class, *args, **kwargs)
        return cls._instance
    
    def _init(self, hwi_class: Type[HardwareInterface], *args, **kwargs):
        """Init the hardware interface
        
        Parameters
        ----------
        hwi_class
            The hardware interface class to init
        """
        
        self.hwi = hwi_class(*args, **kwargs)
    
    def set_light(self, on: bool = False):
        """Switch light on/off
        
        PARAMETERS
        ----------
        on
            Flag to turn on/off light.
        """
        
        logging.debug('switch_light {}'.format(on))
        self.hwi.set_light_gpio(on)
    
    def get_light_status(self) -> bool:
        """Get state of light.
        
        RETURNS
        -------
        state
            State of light True (on), False (off).
        """
        
        return self.hwi.get_light_gpio()
    
    @staticmethod
    def calc_steps_for_distance(dist: float, radius: float, steps_pre_rev: float) -> int:
        """Calculate number of steps to move distance.
        
        PARAMETERS
        ----------
        dist
            Distance to travel.
        radius
            Radius of transport wheel.
        steps_pre_rev
            Steps per revolution.
            
        RETURNS
        -------
        int
            Number of steps to travel tangential distance.
        """
       
        steps = int(dist / ((2.0 * radius * math.pi) / steps_pre_rev))
        logging.debug('steps: {}'.format(steps))
        return steps
    
    @staticmethod
    def calc_stepper_motor_sleep_time(velocity: float, radius: float, steps_pre_rev: float) -> float:
        """Calculate number of steps to move distance.

        PARAMETERS
        ----------
        velocity
            Tangential velocity to travel at.
        radius
            Radius of transport wheel.
        steps_pre_rev
            Steps per revolution.

        RETURNS
        -------
        float
            Stepper motor sleep time (between steps)
        """
        
        ts = (2 * radius * math.pi) / (steps_pre_rev * velocity)
        logging.debug('ts: {}'.format(ts))
        return ts
        
    def set_sheet_move(self, wheel_radius: float, trans: float = 1.0, dist: float = 0.0, vel: float = 0.1):
        """Set sheet movement.
        
        PARAMETERS
        ----------
        wheel_radius
            Radius of transport wheel.
        trans
            Transmission factor.
        dist
            Distance to travel.
        vel
            Linear velocity to travel at.
        """
        
        logging.debug('move_sheet {}'.format(dist))
        self.hwi.set_stepper_motor(
            int(self.calc_steps_for_distance(
                dist=dist, radius=wheel_radius / trans, steps_pre_rev=self.hwi.steps_per_rev
            ) * HardwareInterfacePi.tmc2209_velocity_scaling_compensation(vel)),
            self.calc_stepper_motor_sleep_time(
                velocity=vel, radius=wheel_radius / trans, steps_pre_rev=self.hwi.steps_per_rev)
        )

    def set_sheet_move_async(self, wheel_radius: float, dist: float = 0.0, vel: float = 1.0):
        """Set sheet movement asynchronously.

        PARAMETERS
        ----------
        wheel_radius
            Radius of transport wheel.
        dist
            Distance to travel.
        vel
            Linear velocity to travel at.
        """
    
        logging.debug('move_sheet_async {}'.format(dist))
        self.hwi.set_stepper_motor_async(
            self.calc_steps_for_distance(
                dist=dist, radius=wheel_radius, steps_pre_rev=self.hwi.steps_per_rev
            ),
            self.calc_stepper_motor_sleep_time(
                velocity=vel, radius=wheel_radius, steps_pre_rev=self.hwi.steps_per_rev)
        )

    def get_sheet_status(self) -> bool:
        """Get sheet movement status.
        
        RETURNS
        -------
        bool
            State of sheet: True (moving), False (not moving)
        """
        
        return self.hwi.get_stepper_motor_stat()[0]

def arg_parse() -> argparse.ArgumentParser:
    """Command line argument parsing.
    
    A hierarchical argument parser:
    drop_scan_parser
      type_parser: {scan | camera}
        drop-scan: {do | get} [--dummy]
          do: {move | light | image}
            move: distance wheel_radius
            light: on
            image: light [-d, --destination]
          get: {light, sheet}
        camera: [-d, --destination]
        
    Returns
    -------
    argparse.ArgumentParse
        The argument parser
    """
    
    drop_scan_parser = argparse.ArgumentParser(prog='beeclient.drop_scanner')
    logging_group = drop_scan_parser.add_mutually_exclusive_group()
    logging_group.add_argument('--debug', action='store_true', default=False, help='enable debug logging to console')
    logging_group.add_argument('--info', action='store_true', default=False, help='enable info logging to console')
    drop_scan_parser.add_argument('--dummy', action='store_true', default=False,
                                  help='Run application with hardware dummy.')
    drop_scan_parser.add_argument('--file-log', dest='file_log', type=str, default='', metavar='string',
                                  help='enable logging to file')
    
    camera_preferences = drop_scan_parser.add_argument_group('camera preferences')
    camera_preferences.add_argument('-s', '--size', type=lambda x: tuple(literal_eval(x)), default=(4656, 3496),
                                    metavar='(int, int)',
                                    help='image size (default: %(default)s)')
    camera_preferences.add_argument('-f', '--fps', type=int, default=10, metavar='int',
                                    help='camera frame rate (default: %(default)s)')
    camera_preferences.add_argument('-o', '--fourcc', type=str, default='MJPG', metavar='string',
                                    help='camera four character code setting (default: %(default)s)')
    camera_preferences.add_argument('-b', '--brightness', type=int, default=64, metavar='int',
                                    help='set camera brightness (default: %(default)s)')
    camera_preferences.add_argument('-c', '--contrast', type=int, default=0, metavar='int',
                                    help='set camera contrast (default: %(default)s)')
    camera_preferences.add_argument('-t', '--saturation', type=int, default=64, metavar='int',
                                    help='set camera saturation (default: %(default)s)')
    camera_preferences.add_argument('-u', '--hue', type=int, default=0, metavar='int',
                                    help='set camera hue (default: %(default)s)')
    camera_preferences.add_argument('-g', '--gain', type=int, default=1, metavar='int',
                                    help='set camera gain (default: %(default)s)')
    camera_preferences.add_argument('-n', '--sharpness', type=int, default=2, metavar='int',
                                    help='set camera sharpness (default: %(default)s)')
    camera_preferences.add_argument('-w', '--warmup-time', dest='warmup_time', type=float, default=5.0, metavar='float',
                                    help='camera warmup/adjustment time (e.g. to lighting) before taking a picture  '
                                         '(default: %(default)s)')
    camera_preferences.add_argument('-p', '--preview-time', dest='preview_time', type=float, default=10.0, metavar='float',
                                    help='image preview time after image was taken  (default: %(default)s)')
    camera_preferences.add_argument('-a', '--append-properties', dest='append_properties', action='store_true',
                                    default=False, help='append camera properties to image file name')
    camera_preferences.add_argument('-x', '--camera-type', dest='camera_type', type=str, default='csi',
                                    choices=['csi', 'usb'],
                                    help='connected camera type (default: %(default)s)')
    camera_preferences.add_argument('-q', '--image-quality', dest='image_quality', type=int, default=95,
                                    help='image quality in %% (default: %(default)s)')
    
    io_preferences = drop_scan_parser.add_argument_group('i/o preferences')
    io_preferences.add_argument('--pin-light', dest='pin_light', default=40, metavar='int',
                                help='header pin nr.: light on/off (default: %(default)s)')
    io_preferences.add_argument('--pin-step', dest='pin_step', default=37, metavar='int',
                                help='header pin nr.: step motor (default: %(default)s)')
    io_preferences.add_argument('--pin-dir', dest='pin_dir', default=38, metavar='int',
                                help='header pin nr.: direction motor (default: %(default)s)')
    io_preferences.add_argument('--pin-n-en', dest='pin_n_en', default=36, metavar='int',
                                help='header pin nr.: not enable motor (default: %(default)s)')
    io_preferences.add_argument('--pin-n-sleep', dest='pin_n_sleep', default=33, metavar='int',
                                help='header pin nr.: not sleep (default: %(default)s)')
    io_preferences.add_argument('--pin-n-reset', dest='pin_n_reset', default=35, metavar='int',
                                help='header pin nr.: not reset (default: %(default)s)')
    io_preferences.add_argument('--motor-steps-per-rev', dest='motor_steps_per_rev', default=3200,
                                metavar='int', type=int,
                                help='steps per revolution of stepper motor (default: %(default)s)')

    type_parsers = drop_scan_parser.add_subparsers(dest=TYPE)
    scan_parser = type_parsers.add_parser(TYPE_SCAN, help='droppings scanning utility')

    cmd_parsers = scan_parser.add_subparsers(dest=CMD)
    do_parser = cmd_parsers.add_parser(CMD_DO, help='hardware control actions')
    
    action_parsers = do_parser.add_subparsers(dest=ACTION)
    distance_parser = action_parsers.add_parser(ACTION_MOVE, help='move droppings sheet')
    distance_parser.add_argument('distance', type=float, metavar='float',
                                 help='distance to move the sheet module [mm]')
    distance_parser.add_argument('wheel_radius', type=float, metavar='float',
                                 help='transport wheel radius of the sheet module [mm]')
    distance_parser.add_argument('--velocity', type=float, default=0.04, metavar='float',
                                 help='velocity to move the sheet module at [m/s] (default: %(default)s m/s)')
    distance_parser.add_argument('--gear-ratio', dest='gear_ratio', type=float, default=6, metavar='float',
                                 help='gear ratio of motor to sheet (default: %(default)s)')

    light_parser = action_parsers.add_parser(ACTION_LIGHT, help='switch light on/off')
    light_parser.add_argument('on', type=lambda x: bool(util.strtobool(x)), default=False, metavar='bool',
                              help='set light on/off (default: %(default)s)')

    image_parser = action_parsers.add_parser(ACTION_IMAGE, help='record image set')
    image_parser.add_argument('light', type=lambda x: bool(util.strtobool(x)), default=True, metavar='bool',
                              help='set light on/off during image capture (default: %(default)s)')
    image_parser.add_argument('-d', '--destination', type=str, default=os.getcwd(), metavar='string',
                              help='destination directory for captured images (default: current work directory)')

    get_parser = cmd_parsers.add_parser(CMD_GET, help='hardware state queries, prints state dictionary')
    get_parser.add_argument('--light', action='store_true', default=False,
                            help='state of light.')
    get_parser.add_argument('--sheet', action='store_true', default=False,
                            help='state of drop sheet.')

    camera_parser = type_parsers.add_parser(TYPE_CAMERA, help='camera utility')
    camera_parser.add_argument('-d', '--destination', type=str, default='', metavar='string',
                               help='destination directory for captured images (default: no directory)')
    
    return drop_scan_parser

def init_logging(args):
    """Set up logging.
    
    Parameters
    ----------
    args
        Command line arguments
    """
    
    log = logging.getLogger()
    log.setLevel(logging.INFO)
    log.propagate = 0
        
    log_formatter = logging.Formatter('%(module)s %(levelname)-8s %(message)s')
    console_logger = logging.StreamHandler()
    console_logger.setFormatter(log_formatter)
    if args.debug:
        console_logger.setLevel(logging.DEBUG)
    elif args.info:
        console_logger.setLevel(logging.INFO)
    else:
        console_logger.setLevel(logging.ERROR)
    logging.getLogger().addHandler(console_logger)

    if args.file_log:
        file_logger = RotatingFileHandler(args.file_log, mode='a', maxBytes=5*1024*1024, backupCount=1, delay=False)
        file_log_formatter = logging.Formatter('%(asctime)s %(module)s %(levelname)-8s %(message)s')
        file_logger.setFormatter(file_log_formatter)
        file_logger.setLevel(logging.DEBUG)
        logging.getLogger().addHandler(file_logger)
    


TYPE = 'type'
TYPE_SCAN = 'scan'
TYPE_CAMERA = 'camera'
CMD = 'cmd'
CMD_DO = 'do'
CMD_GET = 'get'
ACTION = 'action'
ACTION_MOVE = 'move'
ACTION_LIGHT = 'light'
ACTION_IMAGE = 'image'


args = arg_parse().parse_args()
init_logging(args)

if args.debug:
    logging.getLogger().setLevel(logging.DEBUG)

if args.dummy:
    hwi_class = HardwareInterfaceDummy
else:
    hwi_class = HardwareInterfacePi
hwa = HardwareAbstraction(
    hwi_class,
    pin_light=args.pin_light, pin_n_en=args.pin_n_en, pin_n_reset=args.pin_n_reset, pin_n_sleep=args.pin_n_sleep,
    pin_step=args.pin_step, pin_dir=args.pin_dir, mot_steps_per_rev=args.motor_steps_per_rev
)
#camera_parameters = CameraParameters(
    #size=args.size, fps=args.fps, fourcc=args.fourcc,
    #camera_warm_up_time=args.warmup_time,
    #brightness=args.brightness, contrast=args.contrast, saturation=args.saturation, hue=args.hue, gain=args.gain,
    #sharpness=args.sharpness, image_quality=args.image_quality
#)

#if args.camera_type == 'csi':
    #camera_interface = PiCameraInterface
    #args.warmup_time = 0
#elif args.camera_type == 'usb':
    #camera_interface = UvcCameraInterface
#else:
    #raise ValueError('{} is not supported!'.format(args.camera_type))

if args.type == TYPE_SCAN:
    
    if args.cmd == CMD_DO:
        if args.action == ACTION_MOVE:
            hwa.set_sheet_move(wheel_radius=args.wheel_radius / 1000, dist=args.distance / 1000,
                               vel=args.velocity, trans=args.gear_ratio)
        if args.action == ACTION_LIGHT:
            hwa.set_light(on=args.on)
        if args.action == ACTION_IMAGE:
            #image_provider = ImageProvider(camera_interface=camera_interface, camera_parameters=camera_parameters,
                                           #destination_dir=args.destination,
                                           #append_camera_props_filename=args.append_properties)
            hwa.set_light(on=args.light)
            #image_provider.capture_images(destination_dir=args.destination, warm_up=True, preview=False, save=True)
            # make image
            hwa.set_light(on=False)
        
    elif args.cmd == CMD_GET:
        states = dict()
        if args.light:
            states.update(dict(light=hwa.get_light_status()))
        if args.sheet:
            states.update(dict(sheet=hwa.get_sheet_status()))
        logging.info(states)

elif args.type == TYPE_CAMERA:
    #image_provider = ImageProvider(camera_interface=camera_interface, camera_parameters=camera_parameters,
                                   #destination_dir=args.destination,
                                   #append_camera_props_filename=args.append_properties,
                                   #debug=args.debug)

    #if args.debug:
    #    image_provider.run_preview(size=(640, 480), hwa=hwa)
    #else:
    #    image_provider.run_preview(hwa=hwa)
    pass



